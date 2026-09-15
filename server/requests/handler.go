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
	//TODO: read hello
	for {
		errc := make(chan error)
		go func() {
			defer close(errc)
			var first [1]byte
			_, err := io.ReadFull(conn, first[:])
			if err != nil {
				errc <- err
				return
			}
			var dec []byte
			var buf bytes.Buffer
			buf.Write(dec)
			switch proto.RequestKind(first[0]) {
			case proto.KindUploadFile:
				var arg proto.UploadFileArg
				_, err = arg.ReadFrom(conn)
			case proto.KindUploadFilePart:
				var arg proto.UploadFilePartArg
				_, err = arg.ReadFrom(conn)
			}
			if err != nil {
				errc <- err
				return
			}
		}()
		select {
		case err := <-errc:
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
