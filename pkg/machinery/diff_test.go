package machinery_test

import (
	"github.com/kralicky/kit/pkg/machinery"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Diff", func() {
	It("should handle adding new contexts", func() {
		existing, incoming := sampleClusters(1), sampleClusters(1, 2)

		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContext{},
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
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "renamed"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
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
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "renamed"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
					ChangeType:       machinery.ChangeTypeRename,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
		incoming.AuthInfos["renamed"] = incoming.AuthInfos["authInfo2"].DeepCopy()
		delete(incoming.AuthInfos, "authInfo2")
		incoming.Contexts["renamed"].AuthInfo = "renamed"
		diff, err = machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "renamed"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
					ChangeType:       machinery.ChangeTypeRename,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
	})
	It("should handle a user auth change", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.AuthInfos["authInfo2"].ClientCertificateData = []byte("newCert")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
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
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
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
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
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
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
					ChangeType:       machinery.ChangeTypeModify | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffClusterCAChanged | machinery.ComplexDiffServerChanged,
				},
			},
		}))
	})
	It("should handle a server URL and user auth change", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].Server = "newURL"
		incoming.AuthInfos["authInfo2"].ClientCertificateData = []byte("newCert")
		incoming.AuthInfos["authInfo2"].ClientKeyData = []byte("newKey")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
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
		incoming.AuthInfos["authInfo2"].ClientCertificateData = []byte("newCert")
		incoming.AuthInfos["authInfo2"].ClientKeyData = []byte("newKey")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
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
					AffectedIncoming: machinery.NamedContextFrom(incoming.Contexts, "context2"),
					AffectedExisting: machinery.NamedContext{},
					ChangeType:       machinery.ChangeTypeNew | machinery.ChangeTypeComplex,
					Complex:          machinery.ComplexDiffRenameRequired,
				},
				{
					AffectedIncoming: machinery.NamedContext{},
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
					ChangeType:       machinery.ChangeTypeDelete,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
	})
	It("should handle a deleted context", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1)
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff).To(Equal(&machinery.Diff{
			Items: []machinery.DiffItem{
				{
					AffectedIncoming: machinery.NamedContext{},
					AffectedExisting: machinery.NamedContextFrom(existing.Contexts, "context2"),
					ChangeType:       machinery.ChangeTypeDelete,
					Complex:          machinery.ComplexDiffTypeNone,
				},
			},
		}))
	})
})
