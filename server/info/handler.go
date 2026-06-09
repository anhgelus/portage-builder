package info

import (
	"context"
	"io"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/server/files"
	"golang.org/x/crypto/ssh"
)

func HandleChannel(ctx context.Context, chroot *files.Root, ch ssh.Channel, reqs <-chan *ssh.Request) {
	log := common.ContextLogger(ctx)
	go ssh.DiscardRequests(reqs)
	errc := make(chan error, 1)
	go func() {
		_, err := io.Copy(ch, chroot.Info())
		select {
		case <-ctx.Done():
			return
		default:
		}
		errc <- err
	}()
	select {
	case err := <-errc:
		log.Error("writing info", "error", err)
	case <-ctx.Done():
	}
	err := ch.Close()
	if err != nil {
		log.Error("closing", "error", err)
	}
	log.Info("closed", "reason", context.Cause(ctx))
}
