package proto

import (
	"context"
	"io"
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
func (r Response) Send(ctx context.Context, com Communication) error {
	b, err := prepareCommand(string(r.Cmd), r.Args)
	if err != nil {
		return err
	}
	return com.Write(ctx, b)
}

// ServerHandler handles [Request].
type ServerHandler interface {
	HandleBuildRequest(context.Context, Communication, BuildArg) Response
	HandleConfigRequest(context.Context, Communication, CfgArg) Response
	HandleSendRequest(context.Context, Communication, SendArg) Response
	HandlePartRequest(context.Context, Communication, PartArg) Response
}

type Server struct {
	Communication
	ServerHandler
	// MaxRequestSize in bytes.
	MaxRequestSize uint32
}

// NewServer creates a [Server].
func NewServer(com Communication, handler ServerHandler, maxRequestSize uint32) *Server {
	return &Server{com, handler, maxRequestSize}
}

func handle[T any](ctx context.Context, s *Server, cmd Command, fn func(context.Context, Communication, T) Response) Response {
	var arg T
	err := UnmarshalArgs(cmd.Args, &arg)
	if err != nil {
		return NewErrorResponse("invalid command", err)
	}
	return fn(ctx, s, arg)
}

// Handle and dispatch incoming [Request] to the [ServerHandler].
func (s *Server) Handle(ctx context.Context, r io.Reader) error {
	cmd, err := ParseCommand(ctx, r, s.MaxRequestSize)
	if err != nil {
		return NewErrorResponse("invalid command", err).Send(ctx, s)
	}
	var resp Response
	switch RequestCommand(cmd.Cmd) {
	case HeyRequest:
		var arg HeyArg
		err := UnmarshalArgs(cmd.Args, &arg)
		if err != nil {
			resp = NewErrorResponse("invalid command", err)
		} else if Version != arg.Version {
			resp = NewErrorResponse("failed hey", ErrVersionNotSupported)
		} else {
			resp = NewResponse(HoyResponse, HoyArg{arg.Version, s.MaxRequestSize})
		}
	case BuildRequest:
		resp = handle(ctx, s, cmd, s.HandleBuildRequest)
	case CfgRequest:
		resp = handle(ctx, s, cmd, s.HandleConfigRequest)
	case SendRequest:
		resp = handle(ctx, s, cmd, s.HandleSendRequest)
	case PartRequest:
		resp = handle(ctx, s, cmd, s.HandlePartRequest)
	default:
		resp = NewResponse(ErrorResponse, []byte("unknown command"))
	}
	return resp.Send(ctx, s)
}

// SendOK to the client.
func (s *Server) SendOK() {
	NewResponse(OkResponse, struct{}{})
}

// SendDone to the client.
func (s *Server) SendDone() {
	NewResponse(DoneResponse, struct{}{})
}
