// Independent configuration adapter for the optional MAA plugin worker.
package config

import (
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
)

type Directories struct{ HomeDir, FightDir, InfrastDir, ProfilesDir string }

var D = &Directories{}
var Profiles = struct {
	Connection struct {
		Device string `toml:"device"`
	} `toml:"connection"`
}{}

func init() {
	root := os.Getenv("MAA_APP_DIR")
	if root == "" {
		root = os.Getenv("MAA_DEV")
	}
	if root == "" {
		root = "/app"
	}
	D.HomeDir = root
	D.FightDir = filepath.Join(root, "config/maa/tasks")
	D.InfrastDir = filepath.Join(root, "config/maa/infrast")
	D.ProfilesDir = filepath.Join(root, "config/maa/profiles")
	_, _ = toml.DecodeFile(filepath.Join(D.ProfilesDir, "default.toml"), &Profiles)
	if serial := os.Getenv("MAA_DEVICE"); serial != "" {
		Profiles.Connection.Device = serial
	}
}
