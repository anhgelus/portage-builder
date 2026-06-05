package proto

import (
	"bytes"
	"errors"
	"fmt"
	"unicode/utf8"
)

const Version uint = 1

// RequestCommand identifies the command sent by a client.
type RequestCommand string

// Request commands used
const (
	HeyRequest   RequestCommand = "HEY"
	BuildRequest RequestCommand = "BUILD"
	CfgRequest   RequestCommand = "CONFIG"
	SendRequest  RequestCommand = "SEND"
	PartRequest  RequestCommand = "PART"
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

// Send a command.
func Send(cmd string, args any) ([]byte, error) {
	b, err := MarshalArgs(args)
	if err != nil {
		return nil, err
	}
	a := append([]byte(cmd+" "), b...)
	a = append(a, '\r', '\n')
	return a, nil
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

var (
	ErrNotUtf8     = errors.New("not utf8 encoded")
	ErrMissingCRLF = errors.New("missing CRLF")
)

// ParseCommand from raw bytes.
// Return [ErrInvalidCommand] if the given bytes are invalid.
func ParseCommand(b []byte) (command Command, err error) {
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
	return
}

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
