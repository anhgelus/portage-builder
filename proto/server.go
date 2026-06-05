package proto

import (
	"context"
	"encoding/binary"
	"errors"
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
	HandleBuildRequest(context.Context, BuildArg) Response
	HandleConfigRequest(context.Context, CfgArg) Response
	HandleSendRequest(context.Context, SendArg) Response
	HandlePartRequest(context.Context, PartArg) Response
}

type Server struct {
	ServerHandler
	// MaxRequestSize in bytes
	MaxRequestSize uint
}

func handle[T any](ctx context.Context, cmd Command, fn func(context.Context, T) Response) Response {
	var arg T
	err := UnmarshalArgs(cmd.Args, &arg)
	if err != nil {
		return NewErrorResponse("invalid command", err)
	}
	return fn(ctx, arg)
}

// Handle and dispatch incoming [Request] to the [ServerHandler].
func (s *Server) Handle(ctx context.Context, b []byte) Response {
	if len(b) >= int(s.MaxRequestSize) {
		return NewErrorResponse("too long", errors.New("request exceed server's MaxReqSize"))
	}
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
			binary.BigEndian.AppendUint64(nil, uint64(s.MaxRequestSize)))
	case BuildRequest:
		return handle(ctx, cmd, s.HandleBuildRequest)
	case CfgRequest:
		return handle(ctx, cmd, s.HandleConfigRequest)
	case SendRequest:
		return handle(ctx, cmd, s.HandleSendRequest)
	case PartRequest:
		return handle(ctx, cmd, s.HandlePartRequest)
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
