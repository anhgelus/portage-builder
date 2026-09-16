package requests

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"

	"anhgelus.world/portage-builder/proto"
)

func Handle(ctx context.Context, conn *tls.Conn) {
	defer conn.Close()
	var hello proto.MessageRequest[*proto.HelloArg]
	_, err := hello.ReadFrom(conn, proto.C2S)
	if err != nil {
		return
	}
	switch hello.Arg.Version {
	case proto.V1:
	default:
		new(proto.MessageResponse[proto.NothingArg]{Kind: proto.KindNotSupported}).WriteTo(conn)
		return
	}
	userCert := conn.ConnectionState().PeerCertificates[0]
	s := Session{user: userCert}
	for {
		errc := make(chan error, 1)
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
			var resp io.WriterTo
			switch proto.RequestKind(first[0]) {
			case proto.KindUploadFile:
				var arg proto.UploadFileArg
				_, err = arg.ReadFrom(conn)
				if err != nil {
					errc <- err
					return
				}
				resp, err = s.HandleFile(ctx, &arg)
			case proto.KindUploadFilePart:
				var arg proto.UploadFilePartArg
				_, err = arg.ReadFrom(conn)
				if err != nil {
					errc <- err
					return
				}
				resp, err = s.HandleFilePart(ctx, &arg)
			}
			if err != nil {
				errc <- err
				return
			}
			_, err = resp.WriteTo(conn)
			if err != nil {
				errc <- err
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
