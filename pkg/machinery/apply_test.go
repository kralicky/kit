package machinery_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/kralicky/kit/pkg/machinery"
)

var _ = Describe("Apply", func() {
	It("should correctly add clusters", func() {
		existing, incoming := sampleClusters(1), sampleClusters(1, 2)
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly rename a cluster", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["newCluster2"] = incoming.Clusters["cluster2"].DeepCopy()
		delete(incoming.Clusters, "cluster2")
		incoming.Contexts["context2"].Cluster = "newCluster2"
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly rename an authinfo", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.AuthInfos["newAuthInfo2"] = incoming.AuthInfos["authInfo2"].DeepCopy()
		delete(incoming.AuthInfos, "authInfo2")
		incoming.Contexts["context2"].AuthInfo = "newAuthInfo2"
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly rename both a cluster and authinfo", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["newCluster2"] = incoming.Clusters["cluster2"].DeepCopy()
		delete(incoming.Clusters, "cluster2")
		incoming.AuthInfos["newAuthInfo2"] = incoming.AuthInfos["authInfo2"].DeepCopy()
		delete(incoming.AuthInfos, "authInfo2")
		incoming.Contexts["context2"].Cluster = "newCluster2"
		incoming.Contexts["context2"].AuthInfo = "newAuthInfo2"
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly rename a cluster, authinfo, and context", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["newCluster2"] = incoming.Clusters["cluster2"].DeepCopy()
		delete(incoming.Clusters, "cluster2")
		incoming.AuthInfos["newAuthInfo2"] = incoming.AuthInfos["authInfo2"].DeepCopy()
		delete(incoming.AuthInfos, "authInfo2")
		incoming.Contexts["newContext2"] = incoming.Contexts["context2"].DeepCopy()
		delete(incoming.Contexts, "context2")
		incoming.Contexts["newContext2"].Cluster = "newCluster2"
		incoming.Contexts["newContext2"].AuthInfo = "newAuthInfo2"
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly delete a context", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1)
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly replace a cluster", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 3)
		incoming.Clusters["cluster3"].Server = existing.Clusters["cluster2"].Server
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly handle a modified cluster CA", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].CertificateAuthorityData = []byte("new-ca-data")
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly handle a modified server URL", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].Server = "https://new-server"
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly handle a modified user auth", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.AuthInfos["authInfo2"] = &api.AuthInfo{
			Username: "new-user",
			Password: "new-password",
		}
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly handle a modified user auth", func() {
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.AuthInfos["authInfo2"] = &api.AuthInfo{
			Username: "new-user",
			Password: "new-password",
		}
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(incoming))
	})
	It("should correctly handle all fields modified", func() {
		// This diff is interpreted as two items:
		// 1. New cluster with a conflicting name that must be renamed
		// 2. Old cluster deleted
		existing, incoming := sampleClusters(1, 2), sampleClusters(1, 2)
		incoming.Clusters["cluster2"].Server = "https://new-server"
		incoming.Clusters["cluster2"].CertificateAuthorityData = []byte("new-ca-data")
		incoming.AuthInfos["authInfo2"] = &api.AuthInfo{
			Username: "new-user",
			Password: "new-password",
		}
		diff, err := machinery.ComputeDiff(existing, incoming)
		Expect(err).NotTo(HaveOccurred())
		Expect(diff.Apply(existing, incoming, machinery.AutoResolver)).To(Succeed())
		Expect(existing).To(Equal(&api.Config{
			Clusters: map[string]*api.Cluster{
				"cluster1":   incoming.Clusters["cluster1"],
				"cluster2-1": incoming.Clusters["cluster2"],
			},
			AuthInfos: map[string]*api.AuthInfo{
				"authInfo1":   incoming.AuthInfos["authInfo1"],
				"authInfo2-1": incoming.AuthInfos["authInfo2"],
			},
			Contexts: map[string]*api.Context{
				"context1": incoming.Contexts["context1"],
				"context2-1": {
					Cluster:  "cluster2-1",
					AuthInfo: "authInfo2-1",
				},
			},
		}))
	})
})
