package requests

import (
	"context"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/proto"
)

func HandleChannel(ctx context.Context, srv *proto.Server) {
	log := common.ContextLogger(ctx)
	go func() {
		for {
			// everything is synchrone here, because a channel is only used by one connection
			err := srv.Handle(ctx, nil, nil)
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
	log.Info("closed", "reason", context.Cause(ctx))
}

func replyError(ctx context.Context, err error) error {
	return reply(ctx, proto.NewErrorResponse("invalid request", err))
}

func reply(ctx context.Context, resp proto.Response) error {
	return resp.Send(ctx, nil)
}
