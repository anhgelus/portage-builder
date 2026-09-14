package info

import (
	"context"
	"io"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/server/files"
)

func HandleChannel(ctx context.Context, chroot *files.Root) {
	log := common.ContextLogger(ctx)
	errc := make(chan error, 1)
	go func() {
		_, err := io.Copy(nil, chroot.Info())
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
	log.Info("closed", "reason", context.Cause(ctx))
}
