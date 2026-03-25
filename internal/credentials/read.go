package credentials

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func getCredentialsDirectory() (string, error) {
	value, exists := os.LookupEnv("CREDENTIALS_DIRECTORY")
	if !exists {
		return "", errors.New("the environment variable CREDENTIALS_DIRECTORY is not set")
	}
	if value == "" {
		return "", errors.New("the environment variable CREDENTIALS_DIRECTORY contains an empty string")
	}
	data, err := os.Stat(value)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("the path referenced by CREDENTIALS_DIRECTORY = %s does not exist", value)
	} else if err != nil {
		return "", err
	}

	if !data.IsDir() {
		return "", fmt.Errorf("the path referenced by CREDENTIALS_DIRECTORY = %s is not a directory", value)
	}

	return value, nil
}

func ReadCredential(name string) (string, error) {
	credentialsDirectoryPath, err := getCredentialsDirectory()
	if err != nil {
		return "", err
	}

	root, err := os.OpenRoot(credentialsDirectoryPath)
	if err != nil {
		return "", err
	}
	defer root.Close()

	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	s := string(bytes)
	sTrimmed := strings.TrimSpace(s)

	return sTrimmed, nil
}
