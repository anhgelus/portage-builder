package proto

import (
	"crypto/ecdh"
	"crypto/x509"
	"errors"
	"io"
	"regexp"
)

var (
	ErrArgsNumber = errors.New("invalid number of arguments")
)

type Package string

var packageRegexp = regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9-]+$`)

// IsPackage indicates if the string is a valid Gentoo package.
func IsPackage(s string) bool {
	return packageRegexp.MatchString(s)
}

type ErrArg struct {
	Err error
}

func (e ErrArg) Error() string {
	return e.Err.Error()
}

func (e ErrArg) Unwrap() error {
	return e.Err
}

func (e ErrArg) ReadFrom(r io.Reader) (int64, error) {
	var ln [1]byte
	_, err := io.ReadFull(r, ln[:])
	if err != nil {
		return 0, err
	}
	rawErr := make([]byte, 0, ln[0])
	n, err := io.ReadFull(r, rawErr)
	n += 1
	if err != nil {
		return int64(n), err
	}
	e.Err = errors.New(string(rawErr))
	return int64(n), nil
}

func (e ErrArg) WriteTo(w io.Writer) (int64, error) {
	err := e.Err.Error()
	n, er := w.Write(append([]byte{byte(len(err))}, []byte(err)...))
	return int64(n), er
}

type HelloArg struct {
	Version Version
	Key     *ecdh.PublicKey
}

func (arg *HelloArg) ReadFrom(r io.Reader) (int64, error) {
	var version [1]byte
	_, err := io.ReadFull(r, version[:])
	if err != nil {
		return 0, err
	}
	arg.Version = Version(version[0])
	var keyLength [1]byte
	_, err = io.ReadFull(r, keyLength[:])
	if err != nil {
		return 1, err
	}
	raw := make([]byte, 0, keyLength[0])
	n, err := io.ReadFull(r, raw)
	ln := 2 + int64(n)
	if err != nil {
		return ln, err
	}
	rawKey, err := x509.ParsePKIXPublicKey(raw)
	if err != nil {
		return ln, ErrArg{err}
	}
	var ok bool
	arg.Key, ok = rawKey.(*ecdh.PublicKey)
	if !ok {
		return ln, ErrArg{errors.New("not an ECDH public key")}
	}
	return ln, nil
}

func (arg *HelloArg) WriteTo(w io.Writer) (int64, error) {
	b, err := x509.MarshalPKIXPublicKey(arg.Key)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(append([]byte{byte(arg.Version), byte(len(b))}, b...))
	return int64(n), err
}
