package proto

import (
	"errors"
	"io"
)

var OkResponse = &MessageResponse[NothingArg]{Kind: KindOk}

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

func NewOkResponse[T Arg](arg T) *MessageResponse[T] {
	return &MessageResponse[T]{Kind: KindOk, Arg: arg}
}
