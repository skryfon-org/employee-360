package service

// MaxPasswordBytes is the longest password bcrypt can hash; it ignores (or, in
// newer versions, rejects) anything past 72 bytes, so longer input is refused.
const MaxPasswordBytes = 72

// HashService defines operations for hashing passwords and secret tokens.
type HashService interface {
	// HashPassword generates a secure bcrypt hash of a plaintext password.
	HashPassword(password string) (string, error)

	// ComparePassword checks if a plaintext password matches a bcrypt hashed password.
	ComparePassword(hashedPassword, password string) error

	// HashToken calculates a SHA-256 hash of a plaintext token for safe database storage.
	HashToken(plainToken string) string
}
