package hashing

import (
	"reflect"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestPhcStringFunctions(t *testing.T) {
	config := DefaultParameters()
	salt, err := GenerateSalt(16)
	if err != nil {
		t.Fail()
		return
	}
	config.Salt = salt

	password := []byte("hunter2")
	config.Hash = argon2.IDKey(password, config.Salt, config.Time, config.Memory, config.Threads, config.KeyLength)

	stringified := createPhcString(config)

	parsedConfig, err := parsePhcString(stringified)
	if err != nil {
		t.Log("Failed to parse generated argon2id PHC string.")
		t.Fail()
		return
	}
	if !reflect.DeepEqual(parsedConfig, config) {
		t.Log("Structs are not deeply equal:")
		t.Logf("%v\n compared to\n%v", parsedConfig, config)
		t.Fail()
		return
	}
}
