package hashing

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"golang.org/x/crypto/argon2"
)

type Argon2IdHash struct {
	Hash      []byte
	KeyLength uint32
	Memory    uint32
	Salt      []byte
	Threads   uint8
	Time      uint32
	Version   int
}

func DefaultParameters() *Argon2IdHash {
	return &Argon2IdHash{
		KeyLength: 32,
		Memory:    64 * 1024,
		Threads:   4,
		Time:      3,
		Version:   argon2.Version,
	}
}

func createPhcString(h *Argon2IdHash) string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", h.Version, h.Memory, h.Time, h.Threads, base64.RawStdEncoding.EncodeToString(h.Salt), base64.RawStdEncoding.EncodeToString(h.Hash))
}

func GenerateSalt(length int) ([]byte, error) {
	out := make([]byte, length)
	_, err := rand.Read(out)
	if err != nil {
		return nil, err
	} else {
		return out, nil
	}
}

func parsePhcString(inp string) (*Argon2IdHash, error) {
	compiledRegex := regexp.MustCompile(`^\$argon2id\$v=(\d+)\$m=(\d+),t=(\d+),p=(\d+)\$([-A-Za-z0-9+/]+)\$([-A-Za-z0-9+/]+)$`)
	matches := compiledRegex.MatchString(inp)
	if !matches {
		return nil, errors.New("argon2id string not valid")
	}

	matchesSlice := compiledRegex.FindAllStringSubmatch(inp, 1)
	if len(matchesSlice) != 1 {
		return nil, errors.New("matches slice length is unexpectedly not 1")
	}

	captureGroups := matchesSlice[0]

	if len(captureGroups) != 7 {
		return nil, errors.New("matches slice first element length is unexpectedly not 8")
	}

	returnValue := &Argon2IdHash{
		KeyLength: 32,
	}

	versionParsed, err := strconv.ParseInt(string(captureGroups[1]), 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version number: %v", err)
	}

	if versionParsed != argon2.Version {
		return nil, errors.New("version of parsed hash different than compiled library")
	}

	returnValue.Version = int(versionParsed)

	memoryParsed, err := strconv.ParseUint(string(captureGroups[2]), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse memory parameter: %v", err)
	}

	returnValue.Memory = uint32(memoryParsed)

	timeParsed, err := strconv.ParseUint(string(captureGroups[3]), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time parameter: %v", err)
	}

	returnValue.Time = uint32(timeParsed)

	threadsParsed, err := strconv.ParseUint(string(captureGroups[4]), 10, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to parse threads parameter: %v", err)
	}

	returnValue.Threads = uint8(threadsParsed)

	returnValue.Salt = make([]byte, base64.RawStdEncoding.DecodedLen(len(captureGroups[5])))
	returnValue.Hash = make([]byte, base64.RawStdEncoding.DecodedLen(len(captureGroups[6])))
	_, err = base64.RawStdEncoding.Decode(returnValue.Salt, []byte(captureGroups[5]))
	if err != nil {
		return nil, err
	}

	_, err = base64.RawStdEncoding.Decode(returnValue.Hash, []byte(captureGroups[6]))
	if err != nil {
		return nil, err
	}

	return returnValue, nil
}
