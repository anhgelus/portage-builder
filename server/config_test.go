package server

import (
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

func TestConfig_Default(t *testing.T) {
	var cfg Config
	err := toml.Unmarshal(DefaultConfig, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig(tmpPath())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != defaultPort {
		t.Error("invalid port", cfg.Port)
	}
	if cfg.MaxRequestSize != defaultMaxRequestSize {
		t.Error("invalid max_request_size", cfg.MaxRequestSize)
	}
	if cfg.Keys.Perms != defaultKeysPerms {
		t.Error("invalid keys perms", cfg.Keys.Perms)
	}
	for k, v := range cfg.Users {
		if v.PublicKey != k+"_pub_key" {
			t.Error("invalid user", k, "pub key", v.PublicKey)
		}
		if v.Name != k {
			t.Error("invalid user", k, "name", v.Name)
		}
	}
}

const (
	missingServerKeys    = `port = 2020`
	missingUserPublicKey = `[server_keys]
public_key_file = ""
private_key_file = ""
[users.foo]
name = "Foo"
`
)

func TestConfig_Invalid(t *testing.T) {
	path := tmpPath()
	err := os.WriteFile(path, []byte(missingServerKeys), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadConfig(path)
	if err == nil {
		t.Fatal("expecting error")
	}
	cv, ok := errors.AsType[ErrInvalidConfig](err)
	if !ok {
		t.Fatal("expecting ErrInvalidConfig, not ", err)
	}
	if !slices.Equal(cv.KeysMissing, []string{"server_keys.public_key_file", "server_keys.private_key_file"}) {
		t.Error("invalid missing keys: ", cv.KeysMissing)
	}

	err = os.WriteFile(path, []byte(missingUserPublicKey), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadConfig(path)
	if err == nil {
		t.Fatal("expecting error")
	}
	cv, ok = errors.AsType[ErrInvalidConfig](err)
	if !ok {
		t.Fatal("expecting ErrInvalidConfig, not ", err)
	}
	if !slices.Equal(cv.KeysMissing, []string{"users.foo.public_key"}) {
		t.Error("invalid missing keys: ", cv.KeysMissing)
	}
}

func tmpPath() string {
	return "/tmp/." + time.Now().Format(time.RFC1123Z) + ".config.toml.test"
}
