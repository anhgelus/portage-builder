package server

import (
	_ "embed"
	"errors"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config of the server.
type Config struct {
	Port        uint   `toml:"port"`
	DataFolder  string `toml:"data_folder"`
	UsersFolder string `toml:"users_folder"`
	Stage3      string `toml:"stage3"`
	// MaxRequestSize in Kio.
	MaxRequestSize uint32 `toml:"max_request_size"`
	Keys           Keys   `toml:"keys"`
}

// Keys of the server.
type Keys struct {
	Root   Key `toml:"root"`
	Server Key `toml:"server"`
}

type Key struct {
	PemFile        string `toml:"cert"`
	PrivateKeyFile string `toml:"private_key"`
}

func (k Key) Read() (cert []byte, key []byte, err error) {
	cert, err = os.ReadFile(k.PemFile)
	if err != nil {
		return
	}
	key, err = os.ReadFile(k.PrivateKeyFile)
	return
}

func (k Key) VerifyPermissions() error {
	info, err := os.Stat(k.PemFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if info.Mode().Perm() != 0o600 {
			return &os.PathError{
				Op:   "checking permission",
				Err:  errors.New("insecured permission"),
				Path: k.PemFile,
			}
		}
	}
	info, err = os.Stat(k.PrivateKeyFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		if info.Mode().Perm() != 0o600 {
			return &os.PathError{
				Op:   "checking permission",
				Err:  errors.New("insecured permission"),
				Path: k.PrivateKeyFile,
			}
		}
	}
	return nil
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

var requiredKeys = [][]string{{"keys", "root"}, {"keys", "server"}}

// Default configs
const (
	defaultPort           uint        = 2020
	defaultMaxRequestSize uint32      = 1024
	defaultDataFolder                 = "/var/lib/portage-builder/"
	defaultUsersFolder                = "user"
	defaultStage3                     = "https://distfiles.gentoo.org/releases/amd64/autobuilds/20260610T214636Z/stage3-amd64-openrc-20260610T214636Z.tar.xz"
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
		DataFolder:     defaultDataFolder,
		UsersFolder:    defaultUsersFolder,
		MaxRequestSize: defaultMaxRequestSize,
		Stage3:         defaultStage3,
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
	if len(missing) != 0 {
		return cfg, ErrInvalidConfig{missing}
	}
	cfg.MaxRequestSize *= 1024
	err = cfg.Keys.Root.VerifyPermissions()
	if err != nil {
		return cfg, err
	}
	err = cfg.Keys.Server.VerifyPermissions()
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}
