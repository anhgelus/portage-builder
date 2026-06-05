package proto

import (
	"context"
	"crypto/sha3"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

type dummyServer struct {
	res any
}

var dummyOk = Response{Cmd: OkResponse, Args: struct{}{}}

func (s *dummyServer) HandleBuildRequest(_ context.Context, _ Communication, arg BuildArg) Response {
	s.res = arg
	return dummyOk
}

func (s *dummyServer) HandleConfigRequest(_ context.Context, _ Communication, arg CfgArg) Response {
	s.res = arg
	return dummyOk
}

func (s *dummyServer) HandleSendRequest(_ context.Context, _ Communication, arg SendArg) Response {
	s.res = arg
	return dummyOk
}

func (s *dummyServer) HandlePartRequest(_ context.Context, _ Communication, arg PartArg) Response {
	s.res = arg
	return dummyOk
}

func genPackage() *rapid.Generator[*Package] {
	return rapid.Custom(func(t *rapid.T) *Package {
		gen := rapid.StringOfN(
			rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVXYZ1234567890-")),
			1, -1, -1)
		p := Package(gen.Draw(t, "") + "/" + gen.Draw(t, ""))
		return &p
	})
}

type request struct {
	cmd  RequestCommand
	body []byte
	want any
}

func genRequest() *rapid.Generator[request] {
	return rapid.Custom(func(t *rapid.T) request {
		cmd := rapid.OneOf(
			rapid.Just(HeyRequest),
			rapid.Just(BuildRequest),
			rapid.Just(CfgRequest),
			rapid.Just(SendRequest),
			rapid.Just(PartRequest)).Draw(t, "cmd")
		var args any
		switch cmd {
		case HeyRequest:
			args = HeyArg{1}
		case BuildRequest:
			p := rapid.SliceOfN(genPackage(), 1, -1).Draw(t, "packages")
			args = BuildArg{p}
		case CfgRequest:
			nbr := rapid.Uint8Min(1).Draw(t, "nbr")
			args = CfgArg{nbr}
		case SendRequest:
			path := rapid.StringN(1, -1, -1).Draw(t, "path")
			parts := rapid.UintMin(1).Draw(t, "parts")
			sum := sha3.Sum512(rapid.SliceOf(rapid.Byte()).Draw(t, "checksum"))
			args = SendArg{path, parts, sum}
		case PartRequest:
			part := rapid.UintMin(1).Draw(t, "part")
			content := rapid.SliceOfN(rapid.Byte(), 1, -1).Draw(t, "content")
			args = PartArg{part, uint(len(content)), content}
		}
		b, err := prepareCommand(string(cmd), args)
		if err != nil {
			panic(err)
		}
		return request{cmd, b, args}
	})
}

func TestServer_Handle(t *testing.T) {
	com := newDummyCom()
	s := &Server{com, &dummyServer{}, 1024 * 1024 * 1024}
	defer s.Close()
	rapid.Check(t, func(t *rapid.T) {
		req := genRequest().Draw(t, "req")
		err := s.Handle(t.Context(), req.body)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := ParseCommand(<-com.out)
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
