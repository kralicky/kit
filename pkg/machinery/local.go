package machinery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
)

type LocalData struct {
	Config *api.Config
}

type RemoteCache struct {
	Latest  api.Config   `json:"latest"`
	History []api.Config `json:"history"`
}

func InitRemote(client *RemoteClient) error {
	// Check if the remote cache exists, if not write an empty one
	if _, err := os.Stat(RemoteCachePath()); err != nil {
		// Write the empty cache
		if err := (&RemoteCache{}).WriteToDisk(); err != nil {
			return err
		}
	} else {
		log.Warn("Remote cache already exists, nothing to do.")
	}

	// Check if we need to create the kit mount in vault
	if exists, err := client.KitMountExists(); err != nil {
		return err
	} else if !exists {
		// Create it
		log.Info("Creating kit mount in remote")
		return client.CreateKitMount()
	}
	log.Warn("Remote kit mount already exists, nothing to do.")
	return nil
}

func RemoteCachePath() string {
	return filepath.Join(DotKitPath(), "remote.yaml")
}

func (cache *RemoteCache) WriteToDisk() error {
	data, err := yaml.Marshal(cache)
	if err != nil {
		return err
	}
	path := RemoteCachePath()
	if _, err := os.Stat(path); err == nil {
		// Make the file readable temporarily
		if err := os.Chmod(path, 0600); err != nil {
			return err
		}
	}
	defer os.Chmod(path, 0400)
	return os.WriteFile(path, data, 0600)
}

func InitLocal(remote string) error {
	conf := &KitConfig{
		RemoteURL: remote,
	}
	dotKit := DotKitPath()
	// If ~/.kit/config.yaml exists, kit is already initialized
	if _, err := os.Stat(KitConfigPath()); err == nil {
		log.Warnf("Local config already exists in %s, nothing to do.", dotKit)
		return nil
	}

	// Create the ~/.kit directory
	if err := os.MkdirAll(dotKit, 0700); err != nil {
		return err
	}

	// Discover local kubeconfig store
	store, err := FindKubeconfigStore()
	if err != nil {
		return err
	}
	conf.KubeconfigPath = store

	if err := conf.WriteToDisk(); err != nil {
		return err
	}

	log.Infof("Initialized local configuration in %s", dotKit)
	return nil
}

func FindKubeconfigStore() (string, error) {
	// If KUBECONFIG is set, check that it points to a single file
	if env := os.Getenv("KUBECONFIG"); env != "" {
		// Paths are separated by ":"
		paths := strings.Split(env, ":")
		if len(paths) > 1 {
			return "", ErrMultipleKubeconfigs
		}
		// If the path does not exist, return an error
		if _, err := os.Stat(paths[0]); err != nil {
			return "", ErrKubeconfigDoesNotExist
		}
		return paths[0], nil
	}

	// Return the default path
	return DefaultKubeconfigPath(), nil
}

func DefaultKubeconfigPath() string {
	path, err := homedir.Expand("~/.kube/config")
	if err != nil {
		panic(err)
	}
	return path
}

func ReadLocalData(conf *KitConfig) (*LocalData, error) {
	path := conf.KubeconfigPath
	// Sanity check to make sure the path exists
	if _, err := os.Stat(path); err != nil {
		return nil, ErrKubeconfigDoesNotExist
	}
	// Unmarshal the config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	localData := &LocalData{
		Config: &api.Config{},
	}
	if err := yaml.Unmarshal(data, localData.Config); err != nil {
		return nil, err
	}
	return localData, nil
}

func RemoteCacheExists() bool {
	_, err := os.Stat(RemoteCachePath())
	return err == nil
}
