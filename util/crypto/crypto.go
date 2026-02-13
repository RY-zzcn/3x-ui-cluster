// Package crypto provides cryptographic utilities for password hashing and verification.
package crypto

import (
	"crypto/rand"
	"math/big"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Password character sets
	passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
)

// GenerateRandomPassword generates a cryptographically secure random password
func GenerateRandomPassword(length int) string {
	password := make([]byte, length)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
		if err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		password[i] = passwordChars[num.Int64()]
	}
	return string(password)
}

// ValidatePasswordStrength checks if password meets minimum security requirements
func ValidatePasswordStrength(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasUpper, hasLower, hasDigit := false, false, false
	for _, c := range password {
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		} else if c >= 'a' && c <= 'z' {
			hasLower = true
		} else if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// HashPasswordAsBcrypt generates a bcrypt hash of the given password.
func HashPasswordAsBcrypt(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPasswordHash verifies if the given password matches the bcrypt hash.
func CheckPasswordHash(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
