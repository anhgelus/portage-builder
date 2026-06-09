package common

import (
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

func TestRingBuffer(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.Uint32Range(1, 2<<20).Draw(t, "size")
		buf := NewRingBuffer(uint(size))
		input := rapid.SliceOfN(rapid.Byte(), 1, -1).Draw(t, "input")
		_, err := buf.Write(input)
		if err != nil {
			t.Fatal(err)
		}
		output := make([]byte, len(input))
		n, err := buf.Read(output)
		if err != nil {
			t.Fatal(err)
		}
		last := input[len(input)-n:]
		if !reflect.DeepEqual(last, output[:n]) {
			t.Errorf("invalid buffer: % x, wanted % x", output[:n], last)
		}
	})
}
