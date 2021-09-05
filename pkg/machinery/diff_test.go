package machinery_test

import (
	"fmt"

	"github.com/kralicky/kit/pkg/machinery"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd/api"
)

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
		conf.AuthInfos[fmt.Sprintf("user%d", id)] = &api.AuthInfo{
			ClientCertificateData: []byte(fmt.Sprintf("user%dClientCert", id)),
			ClientKeyData:         []byte(fmt.Sprintf("user%dClientKey", id)),
		}
		conf.Contexts[fmt.Sprintf("context%d", id)] = &api.Context{
			Cluster:  fmt.Sprintf("cluster%d", id),
			AuthInfo: fmt.Sprintf("user%d", id),
		}
	}
	return conf
}

var _ = Describe("Diff", func() {
	It("should handle adding new contexts", func() {
		existing, incoming := sampleClusters(1), sampleClusters(1, 2)

		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: nil,
					ChangeType:       machinery.ChangeTypeNew,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
	})
	It("should handle a rename", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Contexts["renamed"] = incoming.Contexts["context2"].DeepCopy()
		delete(incoming.Contexts, "context2")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["renamed"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeRename,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
		incoming.Clusters["renamed"] = incoming.Clusters["cluster2"].DeepCopy()
		delete(incoming.Clusters, "cluster2")
		incoming.Contexts["renamed"].Cluster = "renamed"
		diff, err = machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["renamed"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeRename,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
		incoming.AuthInfos["renamed"] = incoming.AuthInfos["user2"].DeepCopy()
		delete(incoming.AuthInfos, "user2")
		incoming.Contexts["renamed"].AuthInfo = "renamed"
		diff, err = machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["renamed"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeRename,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
	})
	It("should handle a user auth change", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.AuthInfos["user2"].ClientCertificateData = []byte("newCert")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeModify | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffUserAuthChanged,
				},
			},
		}))
	})
	It("should handle a cluster CA change", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].CertificateAuthorityData = []byte("newCA")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeModify | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffClusterCAChanged,
				},
			},
		}))
	})
	It("should handle a server URL change", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].Server = "newURL"
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeModify | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffServerChanged,
				},
			},
		}))
	})
	It("should handle a server URL and CA change", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].Server = "newURL"
		incoming.Clusters["cluster2"].CertificateAuthorityData = []byte("newCA")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeModify | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffClusterCAChanged | machinery.ComplexDiffServerChanged,
				},
			},
		}))
	})
	It("should handle a server URL and user auth change", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].Server = "newURL"
		incoming.AuthInfos["user2"].ClientCertificateData = []byte("newCert")
		incoming.AuthInfos["user2"].ClientKeyData = []byte("newKey")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeModify | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffUserAuthChanged | machinery.ComplexDiffServerChanged,
				},
			},
		}))
	})
	It("should handle a server replacement", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		// Server URL stays the same
		incoming.Clusters["cluster2"].CertificateAuthorityData = []byte("newCA")
		incoming.AuthInfos["user2"].ClientCertificateData = []byte("newCert")
		incoming.AuthInfos["user2"].ClientKeyData = []byte("newKey")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: existing.Contexts["context2"],
					ChangeType:       machinery.ChangeTypeReplace,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
	})
	It("should handle a new context with a rename required", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 3)
		incoming.Contexts["context2"] = incoming.Contexts["context3"].DeepCopy()
		delete(incoming.Contexts, "context3")

		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: incoming.Contexts["context2"],
					AffectedExisting: nil,
					ChangeType:       machinery.ChangeTypeNew | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffRenameRequired,
				},
			},
		}))
	})
})
