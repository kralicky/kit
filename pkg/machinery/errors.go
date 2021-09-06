package machinery

import "errors"

var ErrAlreadyInitialized = errors.New("already initialized")

var ErrMultipleKubeconfigs = errors.New("KUBECONFIG environment variable contains more than one path")
var ErrKubeconfigDoesNotExist = errors.New("KUBECONFIG environment variable is set to a nonexistent file")

func IsAlreadyInitialized(err error) bool {
	return errors.Is(err, ErrAlreadyInitialized)
}

var ErrVaultNotInitialized = errors.New("vault is not initialized")
var ErrVaultSealed = errors.New("vault is sealed")
var ErrVaultNoKVMount = errors.New("kv secret engine is not enabled in vault")
var ErrRemoteDataNotFound = errors.New("remote cache does not exist")

func IsNotFound(err error) bool {
	return errors.Is(err, ErrRemoteDataNotFound)
}

var ErrItemAlreadyExists = errors.New("an item with this name already exists")
