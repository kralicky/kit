package machinery

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"sigs.k8s.io/yaml"
)

type KitConfig struct {
	RemoteURL      string `json:"remoteUrl"`
	KubeconfigPath string `json:"kubeconfigPath"`
}

func (c *KitConfig) WriteToDisk() error {
	// Write the config to "$HOME/.kit/config.yaml"
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(KitConfigPath(), data, 0600)
}

func ReadConfig() (*KitConfig, error) {
	data, err := os.ReadFile(KitConfigPath())
	if err != nil {
		return nil, err
	}
	var c KitConfig
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func KitConfigPath() string {
	return filepath.Join(DotKitPath(), "config.yaml")
}

func DotKitPath() string {
	path, err := homedir.Expand("~/.kit")
	if err != nil {
		panic(err)
	}
	return path
}
