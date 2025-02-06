package utils

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

func ValidatePassword(password string) bool {
	if len(password) < 6 {
		return false
	}

	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString
		hasNumber  = regexp.MustCompile(`[0-9]`).MatchString
		hasSpecial = regexp.MustCompile(`[!\"#$%&'()*+,\-./:;<=>?@[\\\]^_{|}~]`).MatchString
	)

	return hasUpper(password) && hasLower(password) && hasNumber(password) && hasSpecial(password)
}

// HashPassword hashes the given password and returns the hashed password or an error
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
