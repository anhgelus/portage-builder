package common

import (
	"sync"
)

type RingBuffer struct {
	mu    sync.Mutex
	data  []byte
	next  uint
	size  uint
	count uint
}

func NewRingBuffer(size uint) *RingBuffer {
	return &RingBuffer{data: make([]byte, size), size: size}
}

func (buf *RingBuffer) Write(p []byte) (n int, err error) {
	for _, b := range p {
		buf.data[buf.next] = b
		buf.next = (buf.next + 1) % buf.size
	}
	buf.count = min(buf.count+uint(len(p)), buf.size)
	return len(p), nil
}

func (buf *RingBuffer) Read(p []byte) (n int, err error) {
	n = min(len(p), int(buf.size))
	beg := buf.next + buf.size - buf.count
	for i := range n {
		p[i] = buf.data[(beg+uint(i))%buf.size]
	}
	buf.size -= uint(n)
	buf.count -= uint(n)
	return
}
