package proto

import (
	"encoding/binary"
	"fmt"
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
	HandleBuildRequest(packages []*Package) Response
	HandleConfigRequest(nbr uint8) Response
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
		var arg HeyArg
		err := UnmarshalArgs(cmd.Args, &arg)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		if Version != uint(arg.Version) {
			return NewErrorResponse(
				"unsupported version",
				fmt.Errorf("don't support %v, only %v", arg.Version, Version))
		}
		return NewResponse(
			HoyResponse,
			[]byte{arg.Version},
			binary.BigEndian.AppendUint64(nil, uint64(s.MaxPartLength)))
	case BuildRequest:
		var arg BuildArg
		err := UnmarshalArgs(cmd.Args, &arg)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		return s.HandleBuildRequest(arg.Packages)
	case CfgRequest:
		var arg CfgArg
		err := UnmarshalArgs(cmd.Args, &arg)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		return s.HandleConfigRequest(arg.Files)
	case SendRequest:
		var arg SendArg
		err := UnmarshalArgs(cmd.Args, &arg)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		return s.HandleSendRequest(arg.Path, arg.Parts, arg.Checksum)
	case PartRequest:
		var arg PartArg
		err := UnmarshalArgs(cmd.Args, &arg)
		if err != nil {
			return NewErrorResponse("invalid command", err)
		}
		if arg.Size >= s.MaxPartLength || len(arg.Content) != int(arg.Size) {
			return NewErrorResponse("invalid command", fmt.Errorf("invalid length"))
		}
		return s.HandlePartRequest(arg.Part, arg.Content)
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
