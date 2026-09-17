package requests

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/proto"
	"anhgelus.world/portage-builder/server/files"
)

func Handle(ctx context.Context, conn *tls.Conn, rootManager *files.Manager) {
	defer conn.Close()
	lg := common.ContextLogger(ctx)
	var hello proto.MessageRequest[*proto.HelloArg]
	_, err := hello.ReadFrom(conn, proto.C2S)
	if err != nil {
		lg.Debug("invalid hello", "error", err)
		return
	}
	switch hello.Arg.Version {
	case proto.V1:
	default:
		new(proto.MessageResponse[proto.NothingArg]{Kind: proto.KindNotSupported}).WriteTo(conn)
		return
	}
	userCert := conn.ConnectionState().PeerCertificates[0]
	lg = lg.With("user", userCert.Subject.CommonName)
	ctx = common.WithLogger(ctx, lg)
	chroot, err := rootManager.GetUser(ctx, userCert.Subject.CommonName)
	if err != nil {
		lg.Error("cannot get chroot", "error", err)
		proto.NewErrorResponse(errors.New("internal error")).WriteTo(conn)
		return
	}
	defer chroot.Close(context.Background())
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
			case proto.KindAddPackage:
				var arg proto.ListPackage
				_, err = arg.ReadFrom(conn)
				if err != nil {
					errc <- err
					return
				}
				resp, err = s.HandleAddPackages(ctx, &arg, chroot)
			case proto.KindRemovePackage:
				var arg proto.ListPackage
				_, err = arg.ReadFrom(conn)
				if err != nil {
					errc <- err
					return
				}
				resp, err = s.HandleRemovePackages(ctx, &arg, chroot)
			case proto.KindBuildPackage:
				var arg proto.ListPackage
				_, err = arg.ReadFrom(conn)
				if err != nil {
					errc <- err
					return
				}
				resp, err = s.HandleBuildPackages(ctx, &arg, chroot)
			case proto.KindUpdateWorld:
				resp, err = s.HandleUpdateWorld(ctx, chroot)
			case proto.KindListPackage:
				resp, err = s.HandleListPackages(ctx, chroot)
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
				lg.Error("handling request", "error", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
