package requests

import (
	"context"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/proto"
	"golang.org/x/crypto/ssh"
)

func HandleChannel(ctx context.Context, ch ssh.Channel, reqs <-chan *ssh.Request, maxSize uint32) {
	log := common.ContextLogger(ctx)
	log.Debug("new channel")
	go ssh.DiscardRequests(reqs)
	go func() {
		for {
			cmd, err := proto.ParseCommand(ctx, ch, maxSize)
			// everything is synchrone here, because a channel is only used by one connection
			if err == nil {
				err = handleCommand(ctx, cmd)
			}
			if err == nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Warn("invalid request", "error", err)
			err = replyError(ctx, err)
			if err != nil {
				log.Error("cannot reply", "error", err)
			}
		}
	}()
	<-ctx.Done()
	err := ch.Close()
	if err != nil {
		log.Error("closing", "error", err)
	}
	log.Info("closed", "reason", context.Cause(ctx))
}

func handleCommand(ctx context.Context, cmd proto.Command) error {
	return nil
}

func replyError(ctx context.Context, err error) error {
	return reply(ctx, proto.NewErrorResponse("invalid request", err))
}

func reply(ctx context.Context, resp proto.Response) error {
	return resp.Send(ctx, nil)
}
