package proto

import (
	"context"
	"errors"
	"fmt"
)

// Request is a command sent to the server.
//
// See [NewRequest].
type Request struct {
	Cmd  RequestCommand
	Args any
}

// NewRequest creates a [Request].
func NewRequest(cmd RequestCommand, args any) Request {
	return Request{cmd, args}
}

// ErrResponse represents a response command that describes an error.
type ErrResponse struct {
	Request  *Request
	Response *Command
	Details  error
}

func (e ErrResponse) Error() string {
	return "error in response: " + e.Details.Error()
}

func (e ErrResponse) As(err any) bool {
	switch v := err.(type) {
	case *ErrResponse:
		*v = e
		return true
	default:
		return false
	}
}

func (e ErrResponse) Is(err error) bool {
	switch v := err.(type) {
	case ErrResponse:
		return errors.Is(e.Details, v.Details)
	default:
		return false
	}
}

func (e ErrResponse) Unwrap() error {
	return e.Details
}

// Send the [Request].
func (r Request) Send(ctx context.Context, com Communication) (Command, error) {
	b, err := prepareCommand(string(r.Cmd), r.Args)
	if err != nil {
		return Command{}, err
	}
	err = com.Write(ctx, b)
	if err != nil {
		return Command{}, err
	}
	resp, err := com.Read(ctx)
	if err != nil {
		return Command{}, err
	}
	cmd, err := ParseCommand(ctx, resp, MaxResponseLength)
	if err != nil {
		return cmd, err
	}
	if cmd.Cmd == string(ErrorResponse) {
		var arg ErrorArg
		err = UnmarshalArgs(cmd.Args, &arg)
		if err != nil {
			return Command{}, err
		}
		return Command{}, ErrResponse{&r, &cmd, errors.New(arg.Error)}
	}
	return cmd, nil
}

type Client struct {
	Communication
	Version        uint8
	MaxRequestSize uint32
}

// ErrInvalidResponse is returned when a response doesn't follow the protocol.
type ErrInvalidResponse struct {
	Response *Command
	Reason   error
}

func (e ErrInvalidResponse) Error() string {
	return fmt.Sprintf("invalid reponse: %s (%s) %s", e.Response.Cmd, e.Response.Args, e.Reason)
}

func (e ErrInvalidResponse) As(err any) bool {
	switch v := err.(type) {
	case *ErrInvalidResponse:
		*v = e
		return true
	default:
		return false
	}
}

func (e ErrInvalidResponse) Is(err error) bool {
	switch v := err.(type) {
	case ErrInvalidResponse:
		return errors.Is(v.Reason, e.Reason)
	default:
		return false
	}
}

func (e ErrInvalidResponse) Unwrap() error {
	return e.Reason
}

func NewInvalidResponseCommand(cmd *Command, exp ResponseCommand) ErrInvalidResponse {
	return ErrInvalidResponse{cmd, fmt.Errorf("expected %s instead", exp)}
}

// NewClient creates a new [Client].
func NewClient(ctx context.Context, com Communication) (*Client, error) {
	c := &Client{Communication: com}
	cmd, err := NewRequest(HeyRequest, HeyArg{Version}).Send(ctx, com)
	if err != nil {
		return nil, err
	}
	if cmd.Cmd != string(HoyResponse) {
		return nil, NewInvalidResponseCommand(&cmd, HoyResponse)
	}
	var arg HoyArg
	err = UnmarshalArgs(cmd.Args, &arg)
	if err != nil {
		return nil, err
	}
	if arg.Version != uint8(Version) {
		return nil, ErrInvalidResponse{&cmd, ErrVersionNotSupported}
	}
	c.Version = arg.Version
	c.MaxRequestSize = arg.MaxRequestSize
	return c, nil
}

// RequestBuild of [Package].
func (c *Client) RequestBuild(ctx context.Context, arg BuildArg) error {
	cmd, err := NewRequest(BuildRequest, arg).Send(ctx, c)
	if err != nil {
		return err
	}
	if cmd.Cmd != string(OkResponse) {
		return NewInvalidResponseCommand(&cmd, OkResponse)
	}
	return nil
}

// RequestConfig updates the distant config.
func (c *Client) RequestConfig(ctx context.Context, arg CfgArg) error {
	cmd, err := NewRequest(CfgRequest, arg).Send(ctx, c)
	if err != nil {
		return err
	}
	if cmd.Cmd != string(OkResponse) {
		return NewInvalidResponseCommand(&cmd, OkResponse)
	}
	return nil
}

// RequestSend file to the server.
// Returns false if the file is already on the server.
func (c *Client) RequestSend(ctx context.Context, arg SendArg) (bool, error) {
	cmd, err := NewRequest(SendRequest, arg).Send(ctx, c)
	if err != nil {
		return false, err
	}
	switch ResponseCommand(cmd.Cmd) {
	case OkResponse:
		return true, nil
	case DoneResponse:
		return false, nil
	default:
		return false, NewInvalidResponseCommand(&cmd, OkResponse)
	}
}

// RequestPart sends a part of the file.
func (c *Client) RequestPart(ctx context.Context, arg PartArg) error {
	cmd, err := NewRequest(PartRequest, arg).Send(ctx, c)
	if err != nil {
		return err
	}
	if cmd.Cmd != string(OkResponse) {
		return NewInvalidResponseCommand(&cmd, OkResponse)
	}
	return nil
}
