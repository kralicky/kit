package machinery_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestMachinery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machinery Suite")
}

func sampleClusters(ids ...int) *api.Config {
	conf := &api.Config{
		Clusters:  map[string]*api.Cluster{},
		AuthInfos: map[string]*api.AuthInfo{},
		Contexts:  map[string]*api.Context{},
	}
	for _, id := range ids {
		conf.Clusters[fmt.Sprintf("cluster%d", id)] = &api.Cluster{
			Server:                   fmt.Sprintf("https://host%d:6443", id),
			CertificateAuthorityData: []byte(fmt.Sprintf("cluster%dCA", id)),
		}
		conf.AuthInfos[fmt.Sprintf("authInfo%d", id)] = &api.AuthInfo{
			ClientCertificateData: []byte(fmt.Sprintf("user%dClientCert", id)),
			ClientKeyData:         []byte(fmt.Sprintf("user%dClientKey", id)),
		}
		conf.Contexts[fmt.Sprintf("context%d", id)] = &api.Context{
			Cluster:  fmt.Sprintf("cluster%d", id),
			AuthInfo: fmt.Sprintf("authInfo%d", id),
		}
	}
	return conf
}
