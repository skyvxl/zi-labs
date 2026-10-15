package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	// #nosec G501 -- MD5 is explicitly required for key derivation by the lab variant.
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

const (
	fileMagic      = "LAB3CFB\x01"
	saltSize       = 16
	fileHeaderSize = len(fileMagic) + saltSize + aes.BlockSize
)

var errUnlock = errors.New("неверная парольная фраза или повреждённый файл пользователей")

type fileCipher struct {
	salt   [saltSize]byte
	key    [md5.Size]byte
	macKey [sha256.Size]byte
}

func deriveFileCipher(passphrase string, salt [saltSize]byte) *fileCipher {
	material := append(bytes.Clone(salt[:]), []byte(passphrase)...)
	defer clear(material)
	// #nosec G401 -- The variant requires MD5; AES-128 consumes its 16-byte result.
	key := md5.Sum(material)
	// Domain separation keeps the authentication key distinct from the AES key.
	macMaterial := append([]byte("lab3 file authentication\x00"), key[:]...)
	defer clear(macMaterial)
	return &fileCipher{salt: salt, key: key, macKey: sha256.Sum256(macMaterial)}
}

func newFileCipher(passphrase string) (*fileCipher, error) {
	if passphrase == "" {
		return nil, errors.New("парольная фраза не может быть пустой")
	}
	var salt [saltSize]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return nil, err
	}
	return deriveFileCipher(passphrase, salt), nil
}

func (c *fileCipher) tag(data []byte) []byte {
	mac := hmac.New(sha256.New, c.macKey[:])
	_, _ = mac.Write(data) // hash.Hash.Write never returns an error.
	return mac.Sum(nil)
}

func (c *fileCipher) seal(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		return nil, err
	}
	data := make([]byte, fileHeaderSize+len(plaintext), fileHeaderSize+len(plaintext)+sha256.Size)
	copy(data, fileMagic)
	copy(data[len(fileMagic):], c.salt[:])
	iv := data[len(fileMagic)+saltSize : fileHeaderSize]
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	// #nosec G407 -- CFB is required by the variant; HMAC authenticates the entire file.
	stream := cipher.NewCFBEncrypter(block, iv) //nolint:staticcheck // SA1019: CFB is required by the assignment and paired with HMAC.
	stream.XORKeyStream(data[fileHeaderSize:], plaintext)
	return append(data, c.tag(data)...), nil
}

func decryptFile(passphrase string, data []byte) (*fileCipher, []byte, error) {
	if passphrase == "" || len(data) < fileHeaderSize+sha256.Size || string(data[:len(fileMagic)]) != fileMagic {
		return nil, nil, errUnlock
	}
	var salt [saltSize]byte
	copy(salt[:], data[len(fileMagic):len(fileMagic)+saltSize])
	c := deriveFileCipher(passphrase, salt)
	signed := data[:len(data)-sha256.Size]
	if !hmac.Equal(c.tag(signed), data[len(signed):]) {
		c.clear()
		return nil, nil, errUnlock
	}
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		c.clear()
		return nil, nil, err
	}
	plaintext := make([]byte, len(signed)-fileHeaderSize)
	iv := data[len(fileMagic)+saltSize : fileHeaderSize]
	// #nosec G407 -- The HMAC has been verified before CFB decryption.
	stream := cipher.NewCFBDecrypter(block, iv) //nolint:staticcheck // SA1019: CFB is required by the assignment and paired with HMAC.
	stream.XORKeyStream(plaintext, signed[fileHeaderSize:])
	return c, plaintext, nil
}

func (c *fileCipher) clear() {
	clear(c.key[:])
	clear(c.macKey[:])
}
