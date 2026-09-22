package security

import "golang.org/x/crypto/bcrypt"

// HashPassword generates a bcrypt hash of the given plain-text password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a bcrypt hashed password with its possible
// plain-text equivalent. Returns true on match, or an error on failure.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
