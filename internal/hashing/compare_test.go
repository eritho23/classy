package hashing

import (
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestCheckPassword(t *testing.T) {
	password := []byte("password123")
	salt, err := GenerateSalt(16)
	if err != nil {
		t.Fail()
		return
	}

	config := DefaultParameters()
	config.Salt = salt
	config.Hash = argon2.IDKey(password, config.Salt, config.Time, config.Memory, config.Threads, config.KeyLength)

	stringified := createPhcString(config)

	t.Log(stringified)

	pass, err := CheckPassword(stringified, password)
	if err != nil {
		t.Logf("Error from CheckPassword: %v.", err)
		t.Fail()
	}
	if !pass {
		t.Log("Password check does not pass.")
		t.Fail()
	}

	pass, err = CheckPassword(stringified, []byte("incorrect"))
	if err != nil {
		t.Logf("Error from CheckPassword: %v.", err)
		t.Fail()
	}
	if pass {
		t.Log("Password check matches when it should not.")
		t.Fail()
	}

	newSalt, err := GenerateSalt(16)
	if err != nil {
		t.Fail()
		return
	}

	config.Salt = newSalt
	stringified2 := createPhcString(config)

	pass, err = CheckPassword(stringified2, password)
	if err != nil {
		t.Logf("Error from CheckPassword: %v.", err)
		t.Fail()
	}
	if pass {
		t.Log("Salt change leads to pass.")
		t.Fail()
	}

	pass, err = CheckPassword("$argon2id$v=19$m=65536,t=2,p=1$gZiV/M1gPc22ElAH/Jh1Hw$CWOrkoo7oJBQ/iyh7uJ0LO2aLEfrHwTWllSAxT0zRno", []byte("hunter2"))
	if err != nil {
		t.Logf("Error from CheckPassword: %v.", err)
		t.Fail()
	}
	if pass {
		t.Log("Example from PHC fails.")
		t.Fail()
	}
}
