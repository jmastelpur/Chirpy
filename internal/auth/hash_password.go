package auth

import (
	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	// hash the password using the argon2id.CreateHash function.
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	return hash, err
}

func CheckPasswordHash(password, hash string) bool {
	//  Use the argon2id.ComparePasswordAndHash function to compare the password that the user entered in the HTTP request with the password that is stored in the database.
	result, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false
	}
	return result
}
