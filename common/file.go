package common

import (
	"crypto/sha3"
	"io"
	"io/fs"
)

// ChecksumOf returns the sha3 sum of the file.
func ChecksumOf(fs fs.FS, path string) (sum [32]byte, err error) {
	f, err := fs.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return
	}
	return sha3.Sum256(b), nil
}
