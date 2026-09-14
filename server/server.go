package server

import (
	"context"
	"log/slog"
	"net"
	"path"

	"anhgelus.world/portage-builder/server/files"
	"anhgelus.world/portage-builder/server/requests"
)

type SSH struct {
	serverConfig *Config
	rootManager  *files.Manager
}

// New creates a [SSH] server and init new users.
func New(ctx context.Context, config *Config) (*SSH, error) {
	var ssh SSH
	ssh.rootManager = files.NewManager(path.Join(config.DataFolder, config.UsersFolder))
	ssh.serverConfig = config
	return nil, nil
}

func (ssh *SSH) Serve(ctx context.Context, l net.Listener) error {
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				conn.Close()
				continue
			}
			go requests.Handle(ctx, conn)
		}
	}()
	<-ctx.Done()
	return ctx.Err()
}

func handleBuilds(log *slog.Logger, chroot *files.Root, handler *requests.UserHandler) {
	for pkgs := range handler.PackagesAdded() {
		err := chroot.AppendPackage(pkgs...)
		if err != nil {
			log.Error("appending packages", "error", err, "pkgs", pkgs)
		}
	}
}

func handleFiles(log *slog.Logger, chroot *files.Root, handler *requests.UserHandler) {
	for f := range handler.UploadedFiles() {
		err := chroot.WriteFile(f.Path, f.Content, 0o644)
		if err != nil {
			log.Error("writing file", "error", err, "file", f)
		}
	}
}
