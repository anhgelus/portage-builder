package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"anhgelus.world/portage-builder/common"
	"anhgelus.world/portage-builder/server"
)

var (
	config string = server.DefaultConfigPath
)

func init() {
	flag.StringVar(&config, "config", config, "config path")
}

func main() {
	flag.Parse()
	cfg, err := server.LoadConfig(config)
	if err != nil {
		panic(err)
	}
	args := flag.Args()
	lg := slog.Default()
	if len(args) < 1 {
		ctx, stop := signal.NotifyContext(
			context.Background(),
			os.Kill, os.Interrupt, syscall.SIGTERM)
		defer stop()
		ctx = common.WithLogger(ctx, lg)
		srv, err := server.New(ctx, &cfg)
		if err != nil {
			panic(err)
		}
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
		if err != nil {
			panic(err)
		}
		lg.Info("started")
		err = srv.Serve(ctx, l)
		select {
		case <-ctx.Done():
			lg.Info("exiting")
			return
		default:
			panic(err)
		}
	}
	switch args[0] {
	case "gen-keys":
		if len(args) < 2 {
			println("Missing server name.")
			os.Exit(2)
		}
		err := server.GenerateServerKeys(args[1], cfg.Keys.Root, cfg.Keys.Server)
		if err != nil {
			panic(err)
		}
	case "gen-user-keys":
		if len(args) != 2 && len(args) != 4 {
			println("Missing username.")
			os.Exit(2)
		}
		var key server.Key
		if len(args) == 2 {
			key.PemFile = args[1] + ".pem"
			key.PrivateKeyFile = args[1] + ".key"
		} else {
			key.PemFile = args[2]
			key.PrivateKeyFile = args[3]
		}
		err := server.GenerateUserKey(key, args[1], cfg.Keys.Root)
		if err != nil {
			panic(err)
		}
	default:
		println("Unkown command " + args[0])
		os.Exit(1)
	}
}
