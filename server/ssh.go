package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/proto"
	"anhgelus.world/portage-builder/server/files"
	"anhgelus.world/portage-builder/server/info"
	"anhgelus.world/portage-builder/server/requests"
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	serverConfig *Config
	sshConfig    *ssh.ServerConfig
	rootManager  *files.Manager
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
		rootManager:  files.NewManager(path.Join(config.DataFolder, config.UsersFolder)),
	}, nil
}

func (s *SSH) ListenAndServe(ctx context.Context) error {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(int(s.serverConfig.Port)))
	if err != nil {
		return err
	}
	defer l.Close()
	defer func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second)
		defer cancel()
		_ = s.rootManager.Close(ctx)
	}()
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
	err = l.Close()
	if err != nil {
		log.Error("disconnecting", "error", err)
	}
	log.Info("disconnected")
	return context.Cause(ctx)
}

// handle new tcp [net.Conn].
func (s *SSH) handle(ctx context.Context, tcp net.Conn) {
	log := common.ContextLogger(ctx)
	conn, chans, reqs, err := ssh.NewServerConn(
		tcp,
		s.sshConfig)
	if err != nil {
		log.Warn("handshake", "error", err)
		return
	}
	defer conn.Close()
	user := conn.User()
	log = log.With("user", user)
	// setup chroot
	setupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	chroot, err := s.rootManager.GetUser(setupCtx, user)
	if err != nil {
		log.Error("loading chroot", "error", err)
		return
	}
	// setup server
	handler := requests.NewUserHandler(user, chroot)
	srv := proto.NewServer(
		handler,
		s.serverConfig.MaxRequestSize)
	defer srv.Close()
	// setup listeners
	handleBuilds(log, chroot, handler)
	handleFiles(log, chroot, handler)
	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		log = log.With("channel", newChannel.ChannelType())
		log.Debug("new channel")
		ctx = common.NewLoggerContext(ctx, log)
		var ch ssh.Channel
		var reqs <-chan *ssh.Request
		switch newChannel.ChannelType() {
		case common.RequestChannel:
			ch, reqs, err = newChannel.Accept()
			if err == nil {
				go requests.HandleChannel(ctx, srv, ch, reqs)
			}
		case common.InfoChannel:
			ch, reqs, err = newChannel.Accept()
			if err == nil {
				go info.HandleChannel(ctx, chroot, ch, reqs)
			}
		default:
			log.Debug("rejected")
			err = newChannel.Reject(
				ssh.UnknownChannelType,
				fmt.Sprint("only support", common.RequestChannel, common.InfoChannel))
		}
		if err != nil {
			log.Error("handling channel", "error", err)
		}
	}
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
