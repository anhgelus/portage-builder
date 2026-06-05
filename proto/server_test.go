package proto

import (
	"context"
	"crypto/sha3"
	"reflect"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

type dummyServer struct {
	res any
}

func (s *dummyServer) HandleBuildRequest(_ context.Context, arg BuildArg) Response {
	s.res = arg
	return Response{}
}

func (s *dummyServer) HandleConfigRequest(_ context.Context, arg CfgArg) Response {
	s.res = arg
	return Response{}
}

func (s *dummyServer) HandleSendRequest(_ context.Context, arg SendArg) Response {
	s.res = arg
	return Response{}
}

func (s *dummyServer) HandlePartRequest(_ context.Context, arg PartArg) Response {
	s.res = arg
	return Response{}
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
		b, err := Send(string(cmd), args)
		if err != nil {
			panic(err)
		}
		return request{cmd, b, args}
	})
}

func TestServer_Handle(t *testing.T) {
	s := &Server{&dummyServer{}, 1024 * 1024 * 1024}
	rapid.Check(t, func(t *rapid.T) {
		req := genRequest().Draw(t, "req")
		resp := s.Handle(t.Context(), req.body)
		if resp.Cmd == ErrorResponse {
			var jn strings.Builder
			for _, b := range resp.Args {
				jn.WriteString(string(b))
				jn.WriteRune(' ')
			}
			t.Fatal(jn.String())
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
