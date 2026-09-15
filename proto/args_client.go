package proto

import (
	"encoding/binary"
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

type HelloArg struct {
	Version Version
}

func (arg *HelloArg) ReadFrom(r io.Reader) (int64, error) {
	var version [1]byte
	_, err := io.ReadFull(r, version[:])
	if err != nil {
		return 0, err
	}
	arg.Version = Version(version[0])
	return 1, nil
}

func (arg *HelloArg) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte{byte(arg.Version)})
	return int64(n), err
}

type Flags uint8

type UploadFileArg struct {
	Flags    Flags
	Parts    uint8
	Size     uint32
	Checksum [32]byte
	Filename string
}

func (arg *UploadFileArg) ReadFrom(r io.Reader) (int64, error) {
	var buf [39]byte
	n, err := io.ReadFull(r, buf[:])
	if err != nil {
		return int64(n), err
	}
	arg.Flags = Flags(buf[0])
	arg.Parts = buf[1]
	arg.Size = binary.BigEndian.Uint32(buf[2:])
	arg.Checksum = [32]byte(buf[6 : 32+6])
	str := make([]byte, 0, buf[32+6])
	ln, err := io.ReadFull(r, str)
	arg.Filename = string(str)
	return int64(n + ln), err
}

func (arg *UploadFileArg) WriteTo(w io.Writer) (int64, error) {
	b := make([]byte, 0, 39+len(arg.Filename))
	b[0] = byte(arg.Flags)
	b[1] = arg.Parts
	binary.BigEndian.AppendUint32(b, arg.Size)
	b = append(b, arg.Checksum[:]...)
	b = append(b, []byte(arg.Filename)...)
	n, err := w.Write(b)
	return int64(n), err
}

type UploadFilePartArg struct {
	Part    uint8
	Size    uint16
	Content []byte
}

func (arg *UploadFilePartArg) ReadFrom(r io.Reader) (int64, error) {
	var buf [3]byte
	n, err := io.ReadFull(r, buf[:])
	if err != nil {
		return int64(n), err
	}
	arg.Part = buf[0]
	arg.Size = binary.BigEndian.Uint16(buf[1:])
	arg.Content = make([]byte, 0, arg.Size)
	n, err = io.ReadFull(r, arg.Content)
	if err != nil {
		return int64(n), err
	}
	return int64(3 + n), err
}

func (arg *UploadFilePartArg) WriteTo(w io.Writer) (int64, error) {
	b := make([]byte, 0, 3+len(arg.Content))
	b[0] = arg.Part
	binary.BigEndian.AppendUint16(b, arg.Size)
	b = append(b, arg.Content...)
	n, err := w.Write(b)
	return int64(n), err
}
