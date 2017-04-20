package construct

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"

	"github.com/cespare/xxhash"
	humanize "github.com/dustin/go-humanize"
)

// ErrInvalidPassword is returned when extracting an encrypted password fails.
var ErrInvalidPassword = errors.New("invalid password")

// PasswordBlock is the cipher block used by the Password type to encrypt/decrypt
// a password.
//
// It must be set for the Password type to be functional.
var PasswordBlock cipher.Block

var hashSize = xxhash.New().Size()

// Password implements encrypting and decrypting a password when serialized.
//
// PasswordBlock must be set for the Password type to be functional.
type Password string

var (
	_ encoding.TextMarshaler   = (*Password)(nil)
	_ encoding.TextUnmarshaler = (*Password)(nil)
)

// MarshalText makes Password implement encoding.TextMarshaler.
func (p Password) MarshalText() ([]byte, error) {
	bs := PasswordBlock.BlockSize()

	// <hash of iv+encrypted password><iv><encrypted password>
	buf := make([]byte, hashSize+bs+len(p))

	iv := buf[hashSize : hashSize+bs]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	ciphertext := buf[hashSize+bs:]
	stream := cipher.NewCTR(PasswordBlock, iv)
	stream.XORKeyStream(ciphertext, []byte(p))

	h := xxhash.Sum64(buf[hashSize:])
	binary.LittleEndian.PutUint64(buf, h)

	n := base64.RawStdEncoding.EncodedLen(len(buf))
	encoded := make([]byte, n)
	base64.RawStdEncoding.Encode(encoded, buf)

	return encoded, nil
}

// UnmarshalText makes Password implement encoding.TextUnmarshaler.
func (p *Password) UnmarshalText(text []byte) error {
	n := base64.RawStdEncoding.DecodedLen(len(text))
	buf := make([]byte, n)
	_, err := base64.RawStdEncoding.Decode(buf, text)
	if err != nil {
		return ErrInvalidPassword
	}

	bs := PasswordBlock.BlockSize()
	if len(buf) < hashSize+bs {
		return ErrInvalidPassword
	}

	if xxhash.Sum64(buf[hashSize:]) != binary.LittleEndian.Uint64(buf[:hashSize]) {
		return ErrInvalidPassword
	}

	iv := buf[hashSize : hashSize+bs]
	ciphertext := buf[hashSize+bs:]

	stream := cipher.NewCTR(PasswordBlock, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	*p = Password(ciphertext)

	return nil
}

// BytesSize implements reading and writing bytes sizes.
type BytesSize uint64

var (
	_ encoding.TextMarshaler   = (*BytesSize)(nil)
	_ encoding.TextUnmarshaler = (*BytesSize)(nil)
)

// MarshalText makes BytesSize implement encoding.TextMarshaler.
func (sz BytesSize) MarshalText() ([]byte, error) {
	s := humanize.Bytes(uint64(sz))
	return []byte(s), nil
}

// UnmarshalText makes BytesSize implement encoding.TextUnmarshaler.
func (sz *BytesSize) UnmarshalText(text []byte) error {
	s := string(text)
	u, err := humanize.ParseBytes(s)
	if err == nil {
		*sz = BytesSize(u)
	}
	return err
}
