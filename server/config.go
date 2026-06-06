package server

import (
	_ "embed"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config of the server.
type Config struct {
	Port uint `toml:"port"`
	// MaxRequestSize in Kio.
	MaxRequestSize uint             `toml:"max_request_size"`
	Keys           Keys             `toml:"server_keys"`
	Users          map[string]*User `toml:"users"`
}

// User data
type User struct {
	PublicKey string `toml:"public_key"`
	Name      string `toml:"name"`
}

// Keys of the server.
type Keys struct {
	// Perms used to store the keys.
	Perms          os.FileMode `toml:"permissions"`
	PrivateKeyFile string      `toml:"private_key_file"`
}

//go:embed config.toml
var DefaultConfig []byte

const DefaultConfigPath = "/etc/portage-builderd/config.toml"

type ErrInvalidConfig struct {
	KeysMissing []string
}

func (e ErrInvalidConfig) Error() string {
	var sb strings.Builder
	sb.WriteString("invalid config: keys ")
	ln := len(e.KeysMissing)
	for i, k := range e.KeysMissing {
		sb.WriteString(k)
		switch i {
		case ln - 1:
		case ln - 2:
			sb.WriteString(" and ")
		default:
			sb.WriteString(", ")
		}
	}
	sb.WriteString(" are missing")
	return sb.String()
}

func (e ErrInvalidConfig) As(err any) bool {
	switch cv := err.(type) {
	case *ErrInvalidConfig:
		*cv = e
		return true
	default:
		return false
	}
}

func (e ErrInvalidConfig) Is(err error) bool {
	switch err.(type) {
	case ErrInvalidConfig:
		return true
	default:
		return false
	}
}

var requiredKeys = [][]string{{"server_keys", "private_key_file"}}

const (
	defaultPort           uint        = 2020
	defaultMaxRequestSize uint        = 1024
	defaultKeysPerms      os.FileMode = 0o600
)

// LoadConfig from a path.
// If file doesn't exist, create a new [Config] with [DefaultConfig].
//
// See [DefaultConfigPath].
func LoadConfig(path string) (Config, error) {
	// default value
	cfg := Config{
		Port:           defaultPort,
		MaxRequestSize: defaultMaxRequestSize,
		Keys:           Keys{Perms: defaultKeysPerms},
	}
	mt, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		if !os.IsNotExist(err) {
			return cfg, err
		}
		err = os.WriteFile(path, DefaultConfig, 0o600)
		if err != nil {
			return cfg, err
		}
		// ignoring error because default config is always valid (guaranteed by a test)
		mt, _ = toml.Decode(string(DefaultConfig), &cfg)
	}
	var missing []string
	for _, k := range requiredKeys {
		if !mt.IsDefined(k...) {
			missing = append(missing, strings.Join(k, "."))
		}
	}
	for k, v := range cfg.Users {
		if !mt.IsDefined("users", k, "public_key") {
			missing = append(missing, "users."+k+".public_key")
		}
		if !mt.IsDefined("users", k, "name") {
			v.Name = k
		}
	}
	if len(missing) != 0 {
		return cfg, ErrInvalidConfig{missing}
	}
	return cfg, nil
}
