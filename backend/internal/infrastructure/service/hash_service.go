package service

import (
	"crypto/sha256"
	"encoding/hex"

	domainerrors "github.com/skryfon-org/employee-360/backend/internal/domain/errors"
	domainservice "github.com/skryfon-org/employee-360/backend/internal/domain/service"
	"golang.org/x/crypto/bcrypt"
)

// hashService implements the domainservice.HashService interface.
type hashService struct {
	cost int
}

// NewHashService returns a new instance of HashService.
func NewHashService(cost ...int) domainservice.HashService {
	bcryptCost := bcrypt.DefaultCost
	if len(cost) > 0 && cost[0] >= bcrypt.MinCost && cost[0] <= bcrypt.MaxCost {
		bcryptCost = cost[0]
	}
	return &hashService{
		cost: bcryptCost,
	}
}

// HashPassword generates a bcrypt hash of the given plaintext password.
func (h *hashService) HashPassword(password string) (string, error) {
	if len(password) > domainservice.MaxPasswordBytes {
		return "", domainerrors.ErrInvalidPassword
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ComparePassword compares a bcrypt hashed password with a plaintext candidate.
func (h *hashService) ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// HashToken calculates a SHA-256 hash of a plaintext secret token for safe storage.
func (h *hashService) HashToken(plainToken string) string {
	sum := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(sum[:])
}
