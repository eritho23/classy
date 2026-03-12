package hashing

import (
	"crypto/subtle"

	"golang.org/x/crypto/argon2"
)

func CheckPassword(fromDb string, password []byte) (bool, error) {
	params, err := parsePhcString(fromDb)
	if err != nil {
		return false, err
	}

	newHash := argon2.IDKey(password, params.Salt, params.Time, params.Memory, params.Threads, params.KeyLength)
	oldHash := params.Hash

	match := subtle.ConstantTimeCompare(newHash, oldHash)

	if match == 1 {
		return true, nil
	} else {
		return false, nil
	}
}
