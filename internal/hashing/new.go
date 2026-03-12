package hashing

import "golang.org/x/crypto/argon2"

func GenerateNewHash(password []byte) (string, error) {
	params := DefaultParameters()
	salt, err := GenerateSalt(16)
	if err != nil {
		return "", err
	}

	params.Salt = salt
	params.Hash = argon2.IDKey(password, params.Salt, params.Time, params.Memory, params.Threads, params.KeyLength)

	return createPhcString(params), nil
}
