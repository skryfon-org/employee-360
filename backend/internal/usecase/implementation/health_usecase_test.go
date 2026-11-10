package usecaseimpl

import (
	"context"
	"errors"
	"testing"
)

// fakeDatabasePinger implements service.DatabasePinger for tests.
type fakeDatabasePinger struct {
	err error
}

func (f *fakeDatabasePinger) Ping(ctx context.Context) error {
	return f.err
}

func TestHealthUseCase_Execute_DatabaseOK(t *testing.T) {
	uc := NewHealthUseCase(&fakeDatabasePinger{}, nil)

	result := uc.Execute(context.Background())

	if result.Database != "ok" {
		t.Errorf("expected Database 'ok', got %q", result.Database)
	}
	if result.App == "" {
		t.Error("expected App to be populated")
	}
}

func TestHealthUseCase_Execute_DatabaseUnreachable(t *testing.T) {
	uc := NewHealthUseCase(&fakeDatabasePinger{err: errors.New("connection refused")}, nil)

	result := uc.Execute(context.Background())

	if result.Database != "unreachable" {
		t.Errorf("expected Database 'unreachable', got %q", result.Database)
	}
}

func TestHealthUseCase_Execute_RedisDisabled(t *testing.T) {
	uc := NewHealthUseCase(&fakeDatabasePinger{}, nil)
	if got := uc.Execute(context.Background()).Redis; got != "disabled" {
		t.Errorf("expected Redis 'disabled', got %q", got)
	}
}

func TestHealthUseCase_Execute_RedisOK(t *testing.T) {
	uc := NewHealthUseCase(&fakeDatabasePinger{}, &fakeDatabasePinger{})
	if got := uc.Execute(context.Background()).Redis; got != "ok" {
		t.Errorf("expected Redis 'ok', got %q", got)
	}
}

func TestHealthUseCase_Execute_RedisUnreachable(t *testing.T) {
	uc := NewHealthUseCase(&fakeDatabasePinger{}, &fakeDatabasePinger{err: errors.New("down")})
	result := uc.Execute(context.Background())
	if result.Redis != "unreachable" {
		t.Errorf("expected Redis 'unreachable', got %q", result.Redis)
	}
	if result.Database != "ok" {
		t.Errorf("redis outage must not affect database status, got %q", result.Database)
	}
}
