package utils

import (
	"golang.org/x/crypto/bcrypt"
)

func Encrypt(str string) (string, error) {

	// Generate "hash" to store from user password
	hash, err := bcrypt.GenerateFromPassword([]byte(str), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func HashCompare(str string, hash string) (bool, error) {

	// Comparing the password with the hash
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(str)); err != nil {
		return false, err
	}
	return true, nil
}
