package proto

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Response is a command sent from the server to the client.
//
// See [NewResponse].
type Response struct {
	Cmd  ResponseCommand
	Args [][]byte
}

// NewResponse creates a [Response].
//
// See [NewErrorResponse] to create a new [Reponse] containing an [error].
func NewResponse(cmd ResponseCommand, args ...[]byte) Response {
	return Response{cmd, args}
}

// NewErrorResponse creates a new standard [Response] containing an [error].
//
// See [NewResponse] to create a simple [Response].
func NewErrorResponse(why string, err error) Response {
	return NewResponse(ErrorResponse, []byte(why+":"), []byte(err.Error()))
}

// Send the [Response].
func (r Response) Send() {
	Send(string(r.Cmd), r.Args)
}

// ServerHandler handles [Request].
type ServerHandler interface {
	HandleBuildRequest(packages []string) Response
	HandleConfigRequest(nbr uint) Response
	HandleSendRequest(path string, nbrParts uint, checksum [64]byte) Response
	HandlePartRequest(part uint, content []byte) Response
}

type Server struct {
	ServerHandler
	MaxPartLength uint
}

// Handle and dispatch incoming [Request] to the [ServerHandler].
func (s *Server) Handle(b []byte) Response {
	cmd, err := ParseCommand(b)
	if err != nil {
		return NewErrorResponse("invalid command", err)
	}
	switch RequestCommand(cmd.Cmd) {
	case HeyRequest:
		u, err := strconv.ParseUint(string(cmd.Args), 10, 16)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		if Version != uint(u) {
			return NewErrorResponse(
				"unsupported version",
				fmt.Errorf("don't support %v, only %v", u, Version))
		}
		v := []byte(strconv.Itoa(int(u)))
		return NewResponse(HoyResponse, v, []byte(strconv.Itoa(int(s.MaxPartLength))))
	case BuildRequest:
		pkgs := bytes.Split(cmd.Args, []byte(" "))
		conv := make([]string, len(pkgs))
		for i, pkg := range pkgs {
			if !IsPackage(string(pkg)) {
				return NewErrorResponse(
					"invalid package",
					fmt.Errorf("%q is not a package", pkg))
			}
			conv[i] = string(pkg)
		}
		return s.HandleBuildRequest(conv)
	case CfgRequest:
		nbr := bytes.Split(cmd.Args, []byte(" "))
		if len(nbr) != 1 {
			return NewErrorResponse(
				"invalid command",
				fmt.Errorf("must have 1 arg, not %v", len(nbr)))
		}
		u, err := strconv.ParseUint(string(nbr[0]), 10, 64)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		return s.HandleConfigRequest(uint(u))
	case SendRequest:
		path, next, ok := bytes.Cut(cmd.Args, []byte(" "))
		if !ok {
			return NewErrorResponse(
				"invalid command",
				fmt.Errorf("must have 3 args"))
		}
		if strings.ContainsRune(string(path), ' ') {
			return NewErrorResponse(
				"invalid command",
				fmt.Errorf("path must not have a space"))
		}
		if !utf8.Valid(path) {
			return NewErrorResponse(
				"invalid command",
				fmt.Errorf("the path is not a valid UTF8"))
		}
		var rnbr []byte
		rnbr, next, ok = bytes.Cut(next, []byte(" "))
		if !ok {
			return NewErrorResponse(
				"invalid command",
				fmt.Errorf("must have 3 args"))
		}
		nbr, err := strconv.ParseUint(string(rnbr), 10, 64)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		if len(next) != 64 {
			return NewErrorResponse("invalid command", fmt.Errorf("invalid SHA3-512, % x", next))
		}
		return s.HandleSendRequest(string(path), uint(nbr), [64]byte(next))
	case PartRequest:
		rp, next, ok := bytes.Cut(cmd.Args, []byte(" "))
		if !ok {
			return NewErrorResponse(
				"invalid command",
				fmt.Errorf("must have 3 args"))
		}
		part, err := strconv.ParseUint(string(rp), 10, 64)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		rp, next, ok = bytes.Cut(next, []byte(" "))
		if !ok {
			return NewErrorResponse(
				"invalid command",
				fmt.Errorf("must have 3 args"))
		}
		size, err := strconv.ParseUint(string(rp), 10, 64)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		if len(next) >= int(s.MaxPartLength) || len(next) != int(size) {
			return NewErrorResponse("invalid command", fmt.Errorf("invalid length"))
		}
		return s.HandlePartRequest(uint(part), next)
	default:
		return NewResponse(ErrorResponse, []byte("unknown command"))
	}
}

// SendOK to the client.
func (s *Server) SendOK() {
	NewResponse(OkResponse)
}

// SendDone to the client.
func (s *Server) SendDone() {
	NewResponse(DoneResponse)
}
