package proto

import (
	"bytes"
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
