package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"path"

	"anhgelus.world/portage-builder/server/files"
	"anhgelus.world/portage-builder/server/requests"
)

type Server struct {
	config      *Config
	rootManager *files.Manager
}

// New creates a [Server] server and init new users.
func New(ctx context.Context, config *Config) (*Server, error) {
	var srv Server
	srv.rootManager = files.NewManager(path.Join(config.DataFolder, config.UsersFolder))
	srv.config = config
	return nil, nil
}

func (srv *Server) Serve(ctx context.Context, l net.Listener) error {
	cfg := srv.config
	rootCert, _, err := cfg.Keys.Root.Read()
	if err != nil {
		return err
	}
	serverCert, key, err := cfg.Keys.Server.Read()
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(rootCert) {
		return errors.New("invalid root certificate")
	}
	cert, err := tls.X509KeyPair(serverCert, key)
	if err != nil {
		return err
	}
	l = tls.NewListener(l, &tls.Config{
		ClientCAs:    pool,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				conn.Close()
				continue
			}
			go requests.Handle(ctx, conn.(*tls.Conn), srv.rootManager)
		}
	}()
	<-ctx.Done()
	return ctx.Err()
}
