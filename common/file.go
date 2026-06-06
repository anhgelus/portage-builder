package common

import (
	"crypto/sha3"
	"io"
	"io/fs"
)

// ChecksumOf returns the sha3 sum of the file.
func ChecksumOf(fs fs.FS, path string) ([64]byte, error) {
	f, err := fs.Open(path)
	if err != nil {
		return [64]byte{}, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return [64]byte{}, err
	}
	return sha3.Sum512(b), nil
}
