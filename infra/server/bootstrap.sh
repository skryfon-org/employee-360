#!/usr/bin/env bash
# One-time idempotent bootstrap for the Employee360 EC2 host.
# Supports Ubuntu 24.04+ (apt) and Amazon Linux 2023 (dnf). Quadlets need Podman >= 4.4,
# so Ubuntu 22.04 (Podman 3.4) is NOT supported.
# Run as root: sudo bash bootstrap.sh
# Firewalling is the EC2 security group's job: open 22 (your IP), 80, 443 only.
set -euo pipefail

DEPLOY_USER="deploy"

log() { echo "[bootstrap-employee360] $*"; }
already() { log "  already done: $*"; }

# ── 1. packages ──────────────────────────────────────────────────────────────
log "Installing packages"
if command -v apt-get &>/dev/null; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y podman nginx uidmap slirp4netns passt curl || apt-get install -y podman nginx uidmap slirp4netns curl
elif command -v dnf &>/dev/null; then
  dnf install -y podman nginx chrony
else
  echo "Unsupported OS: need apt-get or dnf" >&2; exit 1
fi

PODMAN_MAJOR_MINOR=$(podman --version | sed -E 's/[^0-9]*([0-9]+)\.([0-9]+).*/\1 \2/')
read -r PM_MAJOR PM_MINOR <<< "$PODMAN_MAJOR_MINOR"
if (( PM_MAJOR < 4 || (PM_MAJOR == 4 && PM_MINOR < 4) )); then
  echo "Podman $(podman --version) is too old for Quadlet (need >= 4.4). Use Ubuntu 24.04 or Amazon Linux 2023." >&2
  exit 1
fi

# ── 2. deploy user ───────────────────────────────────────────────────────────
if id "$DEPLOY_USER" &>/dev/null; then
  already "user $DEPLOY_USER"
else
  log "Creating user $DEPLOY_USER"
  useradd -m -s /bin/bash "$DEPLOY_USER"
fi
DEPLOY_UID=$(id -u "$DEPLOY_USER")

# ── 3. subuid / subgid (rootless podman) ─────────────────────────────────────
if grep -q "^${DEPLOY_USER}:" /etc/subuid; then
  already "/etc/subuid entry"
else
  log "Adding subuid/subgid for $DEPLOY_USER"
  echo "${DEPLOY_USER}:100000:65536" >> /etc/subuid
  echo "${DEPLOY_USER}:100000:65536" >> /etc/subgid
fi

# ── 4. linger: user services must survive SSH logout / reboot ────────────────
if loginctl show-user "$DEPLOY_USER" 2>/dev/null | grep -q "Linger=yes"; then
  already "linger for $DEPLOY_USER"
else
  log "Enabling linger for $DEPLOY_USER"
  loginctl enable-linger "$DEPLOY_USER"
fi

# ── 5. cgroup v2 delegation ──────────────────────────────────────────────────
DELEGATE_CONF="/etc/systemd/system/user@.service.d/delegate.conf"
if [[ -f "$DELEGATE_CONF" ]]; then
  already "cgroup delegation drop-in"
else
  log "Writing cgroup v2 delegation drop-in"
  mkdir -p "$(dirname "$DELEGATE_CONF")"
  printf '[Service]\nDelegate=memory pids cpu io\n' > "$DELEGATE_CONF"
  systemctl daemon-reexec
fi

# ── 6. podman storage migrate after subuid changes ───────────────────────────
log "Starting user manager for $DEPLOY_USER"
systemctl start "user@${DEPLOY_UID}.service"
for _ in $(seq 1 30); do
  [[ -d "/run/user/${DEPLOY_UID}" ]] && break
  sleep 1
done
[[ -d "/run/user/${DEPLOY_UID}" ]] || { echo "/run/user/${DEPLOY_UID} did not appear" >&2; exit 1; }

log "Running podman system migrate for $DEPLOY_USER"
(cd / && sudo -u "$DEPLOY_USER" XDG_RUNTIME_DIR="/run/user/${DEPLOY_UID}" podman system migrate) || true

# ── 7. journald persistent storage ───────────────────────────────────────────
JOURNALD_CONF="/etc/systemd/journald.conf.d/employee360.conf"
if [[ -f "$JOURNALD_CONF" ]]; then
  already "journald config"
else
  log "Configuring persistent journald"
  mkdir -p /var/log/journal "$(dirname "$JOURNALD_CONF")"
  printf '[Journal]\nStorage=persistent\nSystemMaxUse=1G\nMaxRetentionSec=14day\n' > "$JOURNALD_CONF"
  systemctl restart systemd-journald
fi

# ── 8. quadlet + config directories ──────────────────────────────────────────
log "Creating quadlet and config directories"
sudo -u "$DEPLOY_USER" mkdir -p \
  "/home/${DEPLOY_USER}/.config/containers/systemd" \
  "/home/${DEPLOY_USER}/.config/employee360" \
  "/home/${DEPLOY_USER}/.ssh"
chmod 700 "/home/${DEPLOY_USER}/.config/employee360" "/home/${DEPLOY_USER}/.ssh"
sudo -u "$DEPLOY_USER" touch "/home/${DEPLOY_USER}/.ssh/authorized_keys"
chmod 600 "/home/${DEPLOY_USER}/.ssh/authorized_keys"

# ── 9. time sync + web server ────────────────────────────────────────────────
# Ubuntu keeps time via systemd-timesyncd by default; only dnf hosts need chronyd.
# (On Ubuntu chronyd.service is an alias of chrony.service and can't be enabled by that name.)
if command -v dnf &>/dev/null; then
  systemctl enable --now chronyd
fi
systemctl enable --now nginx

log "Bootstrap complete. Next:"
log "  1. Add the deploy SSH public key to /home/${DEPLOY_USER}/.ssh/authorized_keys"
log "  2. Copy infra/server/nginx/employee360.conf to /etc/nginx/conf.d/ (replace the example.com hostnames), nginx -t && systemctl reload nginx"
log "  3. Issue TLS certs (see docs/self-hosting.md) and set the GitHub environment secrets"
