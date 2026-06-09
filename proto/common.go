package proto

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const Version uint8 = 1

// RequestCommand identifies the command sent by a client.
type RequestCommand string

// Request commands used
const (
	HeyRequest    RequestCommand = "HEY"
	BuildRequest  RequestCommand = "PKGBUILD"
	RemoveRequest RequestCommand = "PKGREMOVE"
	UpdateRequest RequestCommand = "PKGUPDATE"
	CfgRequest    RequestCommand = "CONFIG"
	SendRequest   RequestCommand = "SEND"
	PartRequest   RequestCommand = "PART"
)

// ResponseCommand identifies the command sent by the server.
type ResponseCommand string

// Response commands used
const (
	HoyResponse   ResponseCommand = "HOY"
	OkResponse    ResponseCommand = "OK"
	DoneResponse  ResponseCommand = "DONE"
	ErrorResponse ResponseCommand = "ERROR"
)

const MaxResponseLength uint32 = 1024 * 1024

// prepareCommand a command.
func prepareCommand(cmd string, args any) ([]byte, error) {
	b, err := MarshalArgs(args)
	if err != nil {
		return nil, err
	}
	// +3 -> " " and "\r\n"
	ln := uint32(len(b) + len(cmd) + 3)
	var buf bytes.Buffer
	buf.Write(binary.BigEndian.AppendUint32(nil, ln))
	buf.Grow(int(ln + 4))
	buf.WriteString(cmd)
	buf.WriteRune(' ')
	buf.Write(b)
	buf.WriteString("\r\n")
	return buf.Bytes(), nil
}

type Command struct {
	Cmd  string
	Args []byte
}

type ErrInvalidCommand struct {
	given  []byte
	Reason error
}

func (e ErrInvalidCommand) Error() string {
	return fmt.Sprintf("%s (%q)", e.Reason, e.given)
}

func (e ErrInvalidCommand) As(r any) bool {
	switch v := r.(type) {
	case *ErrInvalidCommand:
		*v = e
		return true
	default:
		return false
	}
}

func (e ErrInvalidCommand) Is(r error) bool {
	switch v := r.(type) {
	case ErrInvalidCommand:
		return errors.Is(e.Reason, v.Reason)
	default:
		return false
	}
}

func (e ErrInvalidCommand) Unwrap() error {
	return e.Reason
}

// Errors coming from [ParseCommand].
var (
	ErrNotUtf8          = errors.New("not utf8 encoded")
	ErrMissingCRLF      = errors.New("missing CRLF")
	ErrCannotReadHeader = errors.New("cannot read header")
	ErrRequestTooLong   = errors.New("request is too long to be processed by the server")
)

// ParseCommand from raw bytes.
// Return [ErrInvalidCommand] if the given bytes are invalid.
func ParseCommand(ctx context.Context, r io.Reader, maxSize uint32) (command Command, err error) {
	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		// extract header containing the length of the command
		header := make([]byte, 4)
		_, err = io.ReadFull(r, header)
		if err != nil {
			err = fmt.Errorf("%w: %w", ErrCannotReadHeader, err)
			return
		}
		ln := binary.BigEndian.Uint32(header)
		if ln >= maxSize {
			err = ErrRequestTooLong
			return
		}
		b := make([]byte, ln)
		_, err = io.ReadFull(r, b)
		if err != nil {
			return
		}
		// parse the real command
		nb := bytes.TrimSuffix(b, []byte("\r\n"))
		if len(nb) == len(b) {
			err = ErrInvalidCommand{b, ErrMissingCRLF}
			return
		}
		cmd, args, _ := bytes.Cut(nb, []byte(" "))
		if !utf8.Valid(cmd) {
			err = ErrInvalidCommand{b, ErrNotUtf8}
			return
		}
		command.Cmd = string(cmd)
		command.Args = args
	}()
	select {
	case <-ctx.Done():
		err = context.Cause(ctx)
	case <-done:
	}
	return
}

var ErrVersionNotSupported = errors.New("version not supported")

type HeyArg struct {
	Version uint8
}

type BuildArg struct {
	Packages []*Package
}

type CfgArg struct {
	Files uint8
}

type SendArg struct {
	Path     string
	Parts    uint
	Checksum [64]byte
}

type PartArg struct {
	Part    uint
	Size    uint
	Content []byte
}

type RemoveArg struct {
	Packages []*Package
}

type UpdateArg struct{}

type HoyArg struct {
	Version        uint8
	MaxRequestSize uint32
}

type ErrorArg struct {
	Error string
}
