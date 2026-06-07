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
func (r Response) Send(ctx context.Context, com io.ReadWriteCloser) error {
	b, err := prepareCommand(string(r.Cmd), r.Args)
	if err != nil {
		return err
	}
	errc := make(chan error, 1)
	go func() {
		_, err := com.Write(b)
		errc <- err
	}()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case err := <-errc:
		return err
	}
}

// ServerHandler handles [Request].
type ServerHandler interface {
	Close()
	HandleBuildRequest(context.Context, BuildArg) Response
	HandleConfigRequest(context.Context, CfgArg) Response
	HandleSendRequest(context.Context, SendArg) Response
	HandlePartRequest(context.Context, PartArg) Response
}

type Server struct {
	ServerHandler
	// MaxRequestSize in bytes.
	MaxRequestSize uint32
}

// NewServer creates a [Server].
func NewServer(handler ServerHandler, maxRequestSize uint32) *Server {
	return &Server{handler, maxRequestSize}
}

func handle[T any](
	ctx context.Context,
	s *Server,
	cmd Command,
	fn func(context.Context, T) Response,
) Response {
	arg, err := UnmarshalArgsFor[T](cmd.Args)
	if err != nil {
		return NewErrorResponse("invalid command", err)
	}
	return fn(ctx, arg)
}

// Handle and dispatch incoming [Request] to the [ServerHandler].
func (s *Server) Handle(ctx context.Context, com io.ReadWriteCloser, r io.Reader) error {
	cmd, err := ParseCommand(ctx, r, s.MaxRequestSize)
	if err != nil {
		return NewErrorResponse("invalid command", err).Send(ctx, com)
	}
	var resp Response
	switch RequestCommand(cmd.Cmd) {
	case HeyRequest:
		arg, err := UnmarshalArgsFor[HeyArg](cmd.Args)
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
	return resp.Send(ctx, com)
}

func (s *Server) Close() {
	s.ServerHandler.Close()
}

// SendOK to the client.
func NewOKResponse() Response {
	return NewResponse(OkResponse, struct{}{})
}

// SendDone to the client.
func NewDoneResponse() Response {
	return NewResponse(DoneResponse, struct{}{})
}
