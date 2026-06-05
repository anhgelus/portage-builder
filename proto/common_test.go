package proto

import (
	"bytes"
	"context"
	"crypto/sha3"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func TestParseCommand(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cmd := rapid.StringN(1, -1, -1).Draw(t, "cmd")
		cmd = strings.ReplaceAll(cmd, " ", "-")
		args := rapid.SliceOf(rapid.Byte()).Draw(t, "args")
		args = append(args, '\r', '\n')
		comd, err := ParseCommand(append([]byte(cmd+" "), args...))
		if err != nil {
			t.Fatal(err)
		}
		if comd.Cmd != cmd {
			t.Error("invalid cmd:", comd.Cmd)
		}
		args = args[:len(args)-2]
		if len(comd.Args) == 0 && len(args) == 0 {
			return
		}
		if !bytes.Equal(comd.Args, args) {
			t.Errorf("invalid args: %#v, wanted %#v", comd.Args, args)
		}
	})
}

type dummyCom struct {
	in  chan []byte
	out chan []byte
}

func newDummyCom() *dummyCom {
	in := make(chan []byte, 1)
	out := make(chan []byte, 1)
	return &dummyCom{in, out}
}

func (d *dummyCom) Write(_ context.Context, b []byte) error {
	d.out <- b
	return nil
}

func (d *dummyCom) Read(context.Context) ([]byte, error) {
	return <-d.in, nil
}

func (d *dummyCom) Close() error {
	close(d.in)
	return nil
}

type dummyDualCom struct {
	// out goes from server to client
	server *dummyCom
	// out goes from client to server
	client *dummyCom
}

func setupAutoServer(s *Server, com *dummyDualCom) <-chan error {
	errc := make(chan error, 1)
	// connect client out to server in
	go func() {
		for b := range com.client.out {
			err := s.Handle(context.Background(), b)
			if err != nil {
				errc <- err
			}
		}
		close(errc)
	}()
	// connect server out to client in
	go func() {
		for b := range com.server.out {
			com.client.in <- b
		}
	}()
	return errc
}

func (d *dummyDualCom) Close() error {
	d.server.Close()
	d.client.Close()
	return nil
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
