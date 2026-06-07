package proto

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

type dummyServer struct {
	res any
}

var dummyOk = Response{Cmd: OkResponse, Args: struct{}{}}

func (s *dummyServer) HandleBuildRequest(_ context.Context, arg BuildArg) Response {
	s.res = arg
	return dummyOk
}

func (s *dummyServer) HandleConfigRequest(_ context.Context, arg CfgArg) Response {
	s.res = arg
	return dummyOk
}

func (s *dummyServer) HandleSendRequest(_ context.Context, arg SendArg) Response {
	s.res = arg
	return dummyOk
}

func (s *dummyServer) HandlePartRequest(_ context.Context, arg PartArg) Response {
	s.res = arg
	return dummyOk
}

func (s *dummyServer) Close()

func TestServer_Handle(t *testing.T) {
	com := newDummyCom()
	s := NewServer(&dummyServer{}, 1024*1024)
	defer s.Close()
	rapid.Check(t, func(t *rapid.T) {
		req := genRequest().Draw(t, "req")
		err := s.Handle(t.Context(), com, bytes.NewBuffer(req.body))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := ParseCommand(t.Context(), <-com.out, s.MaxRequestSize)
		if resp.Cmd == string(ErrorResponse) {
			t.Fatal(resp.Args)
		}
		if req.cmd == HeyRequest {
			return
		}
		res := s.ServerHandler.(*dummyServer).res
		if !reflect.DeepEqual(req.want, res) {
			t.Errorf("invalid response: %#v (%T), want %#v (%T)", res, res, req.want, req.want)
		}
	})
}
