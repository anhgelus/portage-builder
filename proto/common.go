package proto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"errors"
	"io"
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
	KindUpdatePackage
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

func DeriveCipher(private *ecdh.PrivateKey, remote *ecdh.PublicKey) (cipher.AEAD, error) {
	secret, err := private.ECDH(remote)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithRandomNonce(block)
}
