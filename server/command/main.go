package main

import (
	"flag"
	"os"

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
	if len(args) < 1 {
		//TODO: launch
		return
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
