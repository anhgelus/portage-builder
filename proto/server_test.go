package proto

import (
	"crypto/sha3"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

type dummyServer struct {
	res any
}

func (s *dummyServer) HandleBuildRequest(packages []string) Response {
	s.res = packages
	return Response{}
}

func (s *dummyServer) HandleConfigRequest(nbr uint) Response {
	s.res = nbr
	return Response{}
}

func (s *dummyServer) HandleSendRequest(path string, nbrParts uint, checksum [64]byte) Response {
	s.res = [3]any{path, nbrParts, checksum}
	return Response{}
}

func (s *dummyServer) HandlePartRequest(part uint, content []byte) Response {
	s.res = [2]any{part, content}
	return Response{}
}

func genPackage() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		gen := rapid.StringOfN(
			rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVXYZ1234567890-")),
			1, -1, -1)
		return gen.Draw(t, "") + "/" + gen.Draw(t, "")
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
		var args [][]byte
		var want any
		switch cmd {
		case HeyRequest:
			args = [][]byte{[]byte("1")}
		case BuildRequest:
			sl := rapid.SliceOfN(genPackage(), 1, -1).Draw(t, "packages")
			args = make([][]byte, len(sl))
			for i, p := range sl {
				args[i] = []byte(p)
			}
			want = sl
		case CfgRequest:
			nbr := rapid.UintMin(1).Draw(t, "nbr")
			args = [][]byte{[]byte(strconv.FormatUint(uint64(nbr), 10))}
			want = nbr
		case SendRequest:
			path := rapid.StringN(1, -1, -1).Draw(t, "path")
			path = strings.ReplaceAll(path, " ", "-")
			parts := rapid.UintMin(1).Draw(t, "parts")
			sum := sha3.Sum512(rapid.SliceOf(rapid.Byte()).Draw(t, "checksum"))
			args = [][]byte{[]byte(path), []byte(strconv.FormatUint(uint64(parts), 10)), sum[:]}
			want = [3]any{path, parts, sum}
		case PartRequest:
			part := rapid.UintMin(1).Draw(t, "part")
			content := rapid.SliceOfN(rapid.Byte(), 1, -1).Draw(t, "content")
			ln := strconv.Itoa(len(content))
			args = [][]byte{[]byte(strconv.FormatUint(uint64(part), 10)), []byte(ln), content}
			want = [2]any{part, content}
		}
		return request{cmd, Send(string(cmd), args), want}
	})
}

func TestServer_Handle(t *testing.T) {
	s := &Server{&dummyServer{}, 1024 * 1024 * 1024}
	rapid.Check(t, func(t *rapid.T) {
		req := genRequest().Draw(t, "req")
		resp := s.Handle(req.body)
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
