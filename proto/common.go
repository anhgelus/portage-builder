package proto

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
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

var packageRegexp = regexp.MustCompile(`^[a-zA-Z0-9-]+/[a-zA-Z0-9-]+$`)

// IsPackage indicates if the string is a valid Gentoo package.
func IsPackage(s string) bool {
	return packageRegexp.MatchString(s)
}

// Send a command.
func Send(cmd string, args [][]byte) []byte {
	a := append([]byte(cmd+" "), bytes.Join(args, []byte(" "))...)
	a = append(a, '\r', '\n')
	return a
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
	return fmt.Sprintf("invalid command: %s (%q)", e.Reason, e.given)
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
	ErrNotUtf8     = errors.New("command is not utf8 encoded")
	ErrMissingCRLF = errors.New("missing CRLF")
)

// ParseCommand from raw bytes.
// Return [ErrInvalidCommand] if the given bytes are invalid.
func ParseCommand(b []byte) (command Command, err error) {
	cmd, args, f := bytes.Cut(b, []byte(" "))
	if !utf8.Valid(cmd) {
		err = ErrInvalidCommand{b, ErrNotUtf8}
		return
	}
	command.Cmd = string(cmd)
	if f {
		command.Args = bytes.TrimSuffix(args, []byte("\r\n"))
		if len(args) == len(command.Args) {
			err = ErrInvalidCommand{b, ErrMissingCRLF}
		}
	} else {
		command.Cmd = strings.TrimSuffix(command.Cmd, "\r\n")
		if len(command.Cmd) == len(cmd) {
			err = ErrInvalidCommand{b, ErrMissingCRLF}
		}
	}
	return
}
