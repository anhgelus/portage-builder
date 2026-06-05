package proto

import (
	"context"
	"errors"
)

// Response is a command sent from the server to the client.
//
// See [NewResponse].
type Response struct {
	Cmd  ResponseCommand
	Args any
}

// NewResponse creates a [Response].
//
// See [NewErrorResponse] to create a new [Reponse] containing an [error].
func NewResponse(cmd ResponseCommand, args any) Response {
	return Response{cmd, args}
}

// NewErrorResponse creates a new standard [Response] containing an [error].
//
// See [NewResponse] to create a simple [Response].
func NewErrorResponse(why string, err error) Response {
	return NewResponse(ErrorResponse, ErrorArg{why + ": " + err.Error()})
}

// Send the [Response].
func (r Response) Send(_ context.Context) error {
	_, err := prepareCommand(string(r.Cmd), r.Args)
	//TODO: send
	return err
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
		if Version != arg.Version {
			return NewErrorResponse("failed hey", ErrVersionNotSupported)
		}
		return NewResponse(HoyResponse, HoyArg{arg.Version, s.MaxRequestSize})
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
	NewResponse(OkResponse, nil)
}

// SendDone to the client.
func (s *Server) SendDone() {
	NewResponse(DoneResponse, nil)
}
