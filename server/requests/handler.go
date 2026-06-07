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
	srv := proto.NewServer(nil, &Server{}, maxSize)
	defer srv.Close()
	go ssh.DiscardRequests(reqs)
	go func() {
		for {
			// everything is synchrone here, because a channel is only used by one connection
			err := srv.Handle(ctx, ch)
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err == nil {
				continue
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

func replyError(ctx context.Context, err error) error {
	return reply(ctx, proto.NewErrorResponse("invalid request", err))
}

func reply(ctx context.Context, resp proto.Response) error {
	return resp.Send(ctx, nil)
}
