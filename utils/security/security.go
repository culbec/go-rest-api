package security

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/argon2"
)

const DEFAULT_TIME uint32 = 5
const DEFAULT_MEMORY uint32 = 7 * 1024
const DEFAULT_THREADS uint8 = 4
const DEFAULT_KEY_LEN uint32 = 32
const DEFAULT_SALT_LEN uint32 = 16

type Argon2idHash struct {
	time    uint32 // number of passes over the memory
	memory  uint32 // memory size in KiB
	threads uint8  // number of threads
	keyLen  uint32 // key length
	saltLen uint32 // salt length
}

type HashSalt struct {
	Hash []byte // hashed password
	Salt []byte // salt used for hashing
}

func NewArgon2idHash(time, memory uint32, threads uint8, keyLen, saltLen uint32) *Argon2idHash {
	return &Argon2idHash{
		time:    time,
		memory:  memory,
		threads: threads,
		keyLen:  keyLen,
		saltLen: saltLen,
	}
}

func secret(len uint32) ([]byte, error) {
	secret_ := make([]byte, len)

	_, err := rand.Read(secret_)
	if err != nil {
		return nil, err
	}

	return secret_, nil
}

func (a *Argon2idHash) GenerateHash(password, salt []byte) (*HashSalt, error) {
	var err error

	if len(salt) == 0 {
		salt, err = secret(a.saltLen)
		if err != nil {
			return nil, err
		}
	}

	hash := hex.EncodeToString(argon2.IDKey(password, salt, a.time, a.memory, a.threads, a.keyLen))
	return &HashSalt{Hash: []byte(hash), Salt: salt}, nil
}

func (a *Argon2idHash) ComparePasswords(password, salt, hash []byte) error {
	hashSalt, err := a.GenerateHash(password, salt)
	if err != nil {
		return err
	}

	if !bytes.Equal(hash, hashSalt.Hash) {
		return errors.New("passwords do not match")
	}

	return nil
}
