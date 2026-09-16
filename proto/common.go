package proto

import (
	"encoding/binary"
	"errors"
	"io"
	"regexp"
)

type Version uint8

const (
	V1 Version = iota + 1
)

type ResponseKind uint8

const (
	KindOk ResponseKind = iota
	KindNotSupported
	KindError
	KindBad
)

type RequestKind uint8

const (
	KindHello RequestKind = iota
	KindUploadFile
	KindUploadFilePart

	KindAddPackage RequestKind = iota + 0x10
	KindRemovePackage
	KindListPackage
	KindBuildPackage
	KindUpdateWorld
)

type Arg interface {
	io.ReaderFrom
	io.WriterTo
}

type NothingArg struct{}

func (arg NothingArg) ReadFrom(io.Reader) (n int64, err error) {
	return
}

func (arg NothingArg) WriteTo(io.Writer) (n int64, err error) {
	return
}

type Package struct{ string }

func NewPackage(raw string) *Package {
	return &Package{raw}
}

func (p *Package) String() string {
	return p.string
}

func (p *Package) ReadFrom(r io.Reader) (int64, error) {
	var ln [2]byte
	n, err := io.ReadFull(r, ln[:])
	if err != nil {
		return int64(n), err
	}
	pack := make([]byte, 0, binary.BigEndian.Uint16(ln[:]))
	n, err = io.ReadFull(r, pack)
	p.string = string(pack)
	return int64(n + 2), err
}

func (p *Package) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 0, 2+len(p.string))
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(p.string)))
	buf = append(buf, []byte(p.string)...)
	n, err := w.Write(buf)
	return int64(n), err
}

var packageRegexp = regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9-]+$`)

// IsPackage indicates if the string is a valid Gentoo package.
func IsPackage(s string) bool {
	return packageRegexp.MatchString(s)
}

type Direction bool

const (
	C2S Direction = false
	S2C Direction = true
)

var (
	ErrNotSupported = errors.New("not supported")
)

type Message[K ~byte, T Arg] struct {
	Kind K
	Arg  T
}

type MessageResponse[T Arg] = Message[ResponseKind, T]
type MessageRequest[T Arg] = Message[RequestKind, T]
type MessageError = MessageResponse[ErrArg]

func (msg *Message[K, T]) ReadFrom(r io.Reader, direction Direction) (int64, error) {
	var kind [1]byte
	_, err := io.ReadFull(r, kind[:])
	if err != nil {
		return 0, err
	}
	msg.Kind = K(kind[0])
	if direction {
		switch ResponseKind(msg.Kind) {
		case KindBad, KindError:
			var arg ErrArg
			n, err := arg.ReadFrom(r)
			return 1 + n, err
		case KindNotSupported:
			return 1, ErrNotSupported
		}
	}
	var arg T
	n, err := arg.ReadFrom(r)
	return 1 + n, err
}

func (msg *Message[K, T]) WriteTo(w io.Writer) (int64, error) {
	_, err := w.Write([]byte{byte(msg.Kind)})
	if err != nil {
		return 0, err
	}
	n, err := msg.Arg.WriteTo(w)
	return 1 + n, err
}
