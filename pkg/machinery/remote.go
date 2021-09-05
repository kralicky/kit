package machinery

import (
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/command/token"
	"sigs.k8s.io/yaml"
)

type RemoteClient struct {
	VaultConfig *vaultapi.Config
	VaultClient *vaultapi.Client
}

func NewRemoteClient(config *KitConfig) (*RemoteClient, error) {
	conf := vaultapi.DefaultConfig()
	conf.Address = config.RemoteURL
	if err := conf.ReadEnvironment(); err != nil {
		return nil, err
	}
	conf.Address = config.RemoteURL
	client, err := vaultapi.NewClient(conf)
	if err != nil {
		return nil, err
	}
	if client.Token() == "" {
		helper, err := token.NewInternalTokenHelper()
		if err != nil {
			return nil, err
		}
		token, err := helper.Get()
		if err != nil {
			return nil, err
		}
		client.SetToken(token)
	}
	return &RemoteClient{
		VaultConfig: conf,
		VaultClient: client,
	}, nil
}

func (r *RemoteClient) CheckConnection() error {
	sys := r.VaultClient.Sys()
	hr, err := sys.Health()
	if err != nil {
		return err
	}
	if !hr.Initialized {
		return ErrVaultNotInitialized
	}
	if hr.Sealed {
		return ErrVaultSealed
	}
	return nil
}

func (r *RemoteClient) KitMountExists() (bool, error) {
	sys := r.VaultClient.Sys()
	mounts, err := sys.ListMounts()
	if err != nil {
		return false, err
	}
	_, ok := mounts["kit/"]
	return ok, nil
}

func (r *RemoteClient) CreateKitMount() error {
	return r.VaultClient.Sys().Mount("kit", &vaultapi.MountInput{
		Type: "kv",
		Config: vaultapi.MountConfigInput{
			Options: map[string]string{
				"version": "2",
			},
		},
	})
}

func (r *RemoteClient) LoadRemoteData() (*RemoteCache, error) {
	logical := r.VaultClient.Logical()

	sec, err := logical.Read("/kit/data")
	if err != nil {
		return nil, err
	}
	if sec == nil || sec.Data == nil {
		return nil, ErrRemoteDataNotFound
	}
	if data, ok := sec.Data["latest"]; ok {
		cache := &RemoteCache{}
		if err := yaml.Unmarshal(data.([]byte), &cache.Latest); err != nil {
			return nil, err
		}
		return cache, nil
	} else {
		return nil, ErrRemoteDataNotFound
	}
}
