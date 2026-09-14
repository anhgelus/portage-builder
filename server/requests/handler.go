package requests

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"errors"
	"io"
	"net"

	"anhgelus.world/portage-builder/proto"
)

func Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	var req proto.MessageRequest[*proto.HelloArg]
	_, err := req.ReadFrom(conn, proto.C2S)
	if err != nil {
		var v *proto.MessageError
		if e, ok := errors.AsType[proto.ErrArg](err); ok {
			v = &proto.MessageError{Kind: proto.KindBad, Arg: e}
		} else {
			v = proto.NewErrorResponse(err)
		}
		v.WriteTo(conn)
		return
	}
	private, err := ecdh.P256().GenerateKey(nil)
	if err != nil {
		proto.NewErrorResponse(err).WriteTo(conn)
		return
	}
	cipher, pubKey, err := proto.DeriveCipher(private, req.Arg.Key)
	if err != nil {
		proto.NewErrorResponse(err).WriteTo(conn)
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
