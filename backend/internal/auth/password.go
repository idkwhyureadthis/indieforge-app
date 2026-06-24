package auth

import "golang.org/x/crypto/bcrypt"

// BcryptHasher implements the Hasher port using bcrypt.
type BcryptHasher struct{}

// Hash bcrypt-hashes a plaintext password for storage.
func (BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// Compare reports whether password matches the given bcrypt hash.
func (BcryptHasher) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
