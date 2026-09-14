package requests

import (
	"bytes"
	"context"
	"io"
	"net"

	"anhgelus.world/portage-builder/proto"
)

func Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	var helloBuf [1]byte
	_, err := io.ReadFull(conn, helloBuf[:])
	if err != nil {
		return
	}
	var keySize [1]byte
	_, err = io.ReadFull(conn, keySize[:])
	if err != nil {
		return
	}
	remote := make([]byte, 0, keySize[0])
	_, err = io.ReadFull(conn, remote)
	if err != nil {
		return
	}
	cipher, pubKey, err := proto.ServerEDCH(proto.Version(helloBuf[0]), remote)
	if err != nil {
		proto.NewErrorResponse("invalid key exchange", err).Send(ctx, conn)
		return
	}
	var buf bytes.Buffer
	buf.WriteByte(byte(len(pubKey)))
	buf.Write(pubKey)
	_, err = buf.WriteTo(conn)
	if err != nil {
		return
	}
	for {
		ch := make(chan []byte)
		go func() {
			b, err := io.ReadAll(conn)
			if err != nil {
				return
			}
			var un []byte
			cipher.Decrypt(un, b)
			ch <- un
		}()
		select {
		case b := <-ch:
			//TODO: handle
		case <-ctx.Done():
			return
		}
	}
}

func replyError(ctx context.Context, err error) error {
	return reply(ctx, proto.NewErrorResponse("invalid request", err))
}

func reply(ctx context.Context, resp proto.Response) error {
	return resp.Send(ctx, nil)
}
