package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"time"

	"anhgelus.world/portage-builder/common"
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	serverConfig *Config
	sshConfig    *ssh.ServerConfig
}

// New creates a [SSH] server.
func New(log *slog.Logger, config *Config) (*SSH, error) {
	algorithms := ssh.SupportedAlgorithms()

	cfg := &ssh.ServerConfig{
		Config: ssh.Config{
			KeyExchanges: algorithms.KeyExchanges,
			Ciphers:      algorithms.Ciphers,
			MACs:         algorithms.MACs,
		},
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			log = log.With("user", conn.User(), "ip", conn.RemoteAddr())
			if err != nil {
				log.Info("auth failed", "error", err)
				return
			}
			log.Info("auth succeeded")
		},
		PublicKeyAuthAlgorithms: algorithms.PublicKeyAuths,
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			id := conn.User()
			user, ok := config.Users[id]
			if !ok {
				time.Sleep(1 * time.Second)
				return nil, fmt.Errorf("invalid connection")
			}
			if subtle.ConstantTimeCompare([]byte(user.PublicKey), key.Marshal()) == 0 {
				return nil, fmt.Errorf("invalid connection")
			}
			return &ssh.Permissions{
				// Record the public key used for authentication.
				Extensions: map[string]string{
					"pubkey-fp": ssh.FingerprintSHA256(key),
				},
			}, nil
		},
	}

	b, err := os.ReadFile(config.Keys.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	private, err := ssh.ParsePrivateKey(b)
	if err != nil {
		return nil, err
	}
	// Restrict host key algorithms to disable ssh-rsa.
	signer, err := ssh.NewSignerWithAlgorithms(
		private.(ssh.AlgorithmSigner),
		[]string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512})
	if err != nil {
		return nil, err
	}
	cfg.AddHostKey(signer)
	return &SSH{
		serverConfig: config,
		sshConfig:    cfg,
	}, nil
}

func (s *SSH) ListenAndServe(ctx context.Context) error {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(int(s.serverConfig.Port)))
	if err != nil {
		return err
	}
	defer l.Close()
	log := common.ContextLogger(ctx)
	//errc := make(chan error, 1)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Warn("accepting connection", "error", err)
				continue
			}
			go s.handle(
				common.NewLoggerContext(ctx, log.With("ip", conn.RemoteAddr())),
				conn)
		}
	}()
	<-ctx.Done()
	return context.Cause(ctx)
}

// handle new tcp [net.Conn].
func (s *SSH) handle(ctx context.Context, tcp net.Conn) {
	log := common.ContextLogger(ctx)
	_, chans, reqs, err := ssh.NewServerConn(
		tcp,
		s.sshConfig)
	if err != nil {
		log.Warn("handshake", "error", err)
		return
	}
	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		log = log.With("channel", newChannel.ChannelType())
		switch newChannel.ChannelType() {
		case common.PkgChannel:
		case common.RequestChannel:
		default:
			err = newChannel.Reject(ssh.UnknownChannelType, "only support pkg and request")
		}
		if err != nil {
			log.Error("handling channel", "error", err)
		}
	}
}
