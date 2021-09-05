package machinery

import (
	"bytes"
	"reflect"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/tools/clientcmd/api"
)

type ChangeType int

const (
	// A new kubeconfig was added and does not interfere with any existing ones
	ChangeTypeNew ChangeType = 1 + iota

	// A kubeconfig was renamed but otherwise has no conflicts
	ChangeTypeRename

	// An existing kubeconfig was removed
	ChangeTypeDelete

	// A new kubeconfig completely replaces an existing one
	ChangeTypeReplace

	// A new kubeconfig completely replaces an existing one
	ChangeTypeModify

	// Some additional things need to be taken care of
	ChangeTypeComplex = 1 << iota
)

type ComplexDiffType int

const (
	ComplexDiffTypeNone ComplexDiffType = 0

	// The server URL has been changed
	ComplexDiffServerChanged ComplexDiffType = 1 << iota

	// The user auth info has changed
	ComplexDiffUserAuthChanged

	// The cluster CA has changed
	ComplexDiffClusterCAChanged

	// The preferences have changed (this field is currently unused)
	ComplexDiffPreferencesChanged

	// The kubeconfig needs to be renamed as it conflicts with an existing one
	ComplexDiffRenameRequired
)

type Diff struct {
	Items []DiffItem
}

type DiffItem struct {
	AffectedIncoming *api.Context
	AffectedExisting *api.Context
	ChangeType       ChangeType
	Complex          ComplexDiffType
	CanAutoMerge     bool
}

func ClustersEqual(a, b *api.Cluster) bool {
	// if both of these are equal, we can be confident that the clusters are equal
	return bytes.Equal(a.CertificateAuthorityData, b.CertificateAuthorityData) &&
		a.Server == b.Server
}

func AuthInfosEqual(a, b *api.AuthInfo) bool {
	return reflect.DeepEqual(a, b)
}

func ComputeIncomingDiff(config *KitConfig, client *RemoteClient) (*Diff, error) {
	local, err := ReadLocalData(config)
	if err != nil {
		return nil, err
	}
	remote, err := ReadRemoteCache()
	if err != nil {
		return nil, err
	}
	localConfig := local.Config
	remoteConfig := remote.Latest
	return ComputeDiff(localConfig, &remoteConfig)
}

func ComputeDiff(existing *api.Config, incoming *api.Config) (*Diff, error) {
	// First pass:
	// 1. First go through each remote kubeconfig and see if there is an exact match
	//    in the local config. If there is, we can skip it.
	// 2. If there is no exact match, check if there is a local match with different
	// 	  names. If there is, mark it as renamed.
	// 3. If there is no exact renamed match, check if there is a local server
	// 		CA match. If there is, mark the user auth, server URL, or name as
	// 		changed if necessary.
	// 4. If there is no server CA match, check if there is a local user auth
	// 		match. If there is, mark the server URL, or CA as changed if necessary.
	// 5. If there is no user auth match, check if there is a local server URL
	// 		match. If there is, mark it as a replacement.
	// 3. Otherwise, it is a new kubeconfig.
	// 3a. If the new kubeconfig has the same name as an existing one, mark it as
	//     RenameRequired

	diff := &Diff{}

CONTEXT:
	for contextName, context := range incoming.Contexts {
		incomingCluster, ok := incoming.Clusters[context.Cluster]
		if !ok {
			// TODO: provide options to fix this automatically
			log.Fatalf("Remote config is ill-formed: context %s references nonexistent cluster %s", contextName, context.Cluster)
		}

		incomingAuth, ok := incoming.AuthInfos[context.AuthInfo]
		if !ok {
			// TODO: provide options to fix this automatically
			log.Fatalf("Remote config is ill-formed: context %s references nonexistent auth info %s", contextName, context.AuthInfo)
		}

		// Check if there is an exact match
		exactMatch := false
		existingContext, ok := existing.Contexts[contextName]
		if ok {
			existingCluster, ok := existing.Clusters[existingContext.Cluster]
			if ok {
				existingAuth, ok := existing.AuthInfos[existingContext.AuthInfo]
				if ok {
					if ClustersEqual(existingCluster, incomingCluster) && AuthInfosEqual(existingAuth, incomingAuth) {
						exactMatch = true
					}
				}
			}
		}

		// If there is an exact match, we can skip it
		if exactMatch {
			continue
		}

		// Check if there is a local match with different names
		var matchingCluster *struct {
			Cluster *api.Cluster
			Context *api.Context
		}
		var matchingAuth *struct {
			AuthInfo *api.AuthInfo
			Context  *api.Context
		}
		for existingContextName, existingContext := range existing.Contexts {
			existingCluster, ok := existing.Clusters[existingContext.Cluster]
			if !ok {
				log.Fatalf("Local config is ill-formed: context %s references nonexistent cluster %s", existingContextName, existingContext.Cluster)
			}
			existingAuth, ok := existing.AuthInfos[existingContext.AuthInfo]
			if !ok {
				log.Fatalf("Local config is ill-formed: context %s references nonexistent auth info %s", existingContextName, existingContext.AuthInfo)
			}
			if ClustersEqual(existingCluster, incomingCluster) {
				matchingCluster = &struct {
					Cluster *api.Cluster
					Context *api.Context
				}{
					Cluster: existingCluster,
					Context: existingContext,
				}
			}
			if AuthInfosEqual(existingAuth, incomingAuth) {
				matchingAuth = &struct {
					AuthInfo *api.AuthInfo
					Context  *api.Context
				}{
					AuthInfo: existingAuth,
					Context:  existingContext,
				}
			}
			if matchingCluster != nil && matchingAuth != nil {
				if matchingCluster.Context != matchingAuth.Context {
					continue
				}
				break
			}
		}
		switch {
		case matchingCluster != nil && matchingAuth != nil:
			if matchingCluster.Context != matchingAuth.Context {
				log.Fatal("???") // TODO: handle this case
				continue
			}
			// Renamed
			diff.Items = append(diff.Items, DiffItem{
				AffectedExisting: matchingCluster.Context,
				AffectedIncoming: context,
				ChangeType:       ChangeTypeRename,
				Complex:          ComplexDiffTypeNone,
			})
			continue
		case matchingCluster != nil:
			// User auth changed
			diff.Items = append(diff.Items, DiffItem{
				AffectedExisting: matchingCluster.Context,
				AffectedIncoming: context,
				ChangeType:       ChangeTypeModify | ChangeTypeComplex,
				Complex:          ComplexDiffUserAuthChanged,
			})
			continue
		case matchingAuth != nil:
			// Cluster and/or server URL changed
			complexType := ComplexDiffTypeNone
			if incomingCluster.Server != existing.Clusters[matchingAuth.Context.Cluster].Server {
				complexType |= ComplexDiffServerChanged
			}
			if !bytes.Equal(incomingCluster.CertificateAuthorityData,
				existing.Clusters[matchingAuth.Context.Cluster].CertificateAuthorityData) {
				complexType |= ComplexDiffClusterCAChanged
			}
			diff.Items = append(diff.Items, DiffItem{
				AffectedExisting: matchingAuth.Context,
				AffectedIncoming: context,
				ChangeType:       ChangeTypeModify | ChangeTypeComplex,
				Complex:          complexType,
			})
			continue
		default:
			// Nothing matched, before assuming it is new, check if the incoming
			// context has a matching server CA. If so, both the user auth and
			// cluster URL are new, but the cluster is not a replacement.
			for existingContextName, existingContext := range existing.Contexts {
				existingCluster, ok := existing.Clusters[existingContext.Cluster]
				if !ok {
					log.Fatalf("Local config is ill-formed: context %s references nonexistent cluster %s", existingContextName, existingContext.Cluster)
				}
				if bytes.Equal(incomingCluster.CertificateAuthorityData,
					existingCluster.CertificateAuthorityData) {
					// Cluster CA is the same, this is a modification
					diff.Items = append(diff.Items, DiffItem{
						AffectedExisting: existingContext,
						AffectedIncoming: context,
						ChangeType:       ChangeTypeModify | ChangeTypeComplex,
						Complex:          ComplexDiffUserAuthChanged | ComplexDiffServerChanged,
					})
					continue CONTEXT
				}
			}
		}

		// Check if there is a local server URL match
		for existingContextName, existingContext := range existing.Contexts {
			existingCluster, ok := existing.Clusters[existingContext.Cluster]
			if !ok {
				log.Fatalf("Local config is ill-formed: context %s references nonexistent cluster %s", existingContextName, existingContext.Cluster)
			}
			if existingCluster.Server == incomingCluster.Server {
				// Replacement
				diff.Items = append(diff.Items, DiffItem{
					AffectedExisting: existingContext,
					AffectedIncoming: context,
					ChangeType:       ChangeTypeReplace,
					Complex:          ComplexDiffTypeNone,
				})
				continue CONTEXT
			}
		}

		var renameRequired ComplexDiffType = ComplexDiffTypeNone
		var changeType = ChangeTypeNew

		// Check if there is a local context name match
		if _, ok := existing.Contexts[contextName]; ok {
			renameRequired = ComplexDiffRenameRequired
		}
		// Check if there is a local cluster name match
		if _, ok := existing.Clusters[context.Cluster]; ok {
			renameRequired = ComplexDiffRenameRequired
		}
		// Check if there is a local auth name match
		if _, ok := existing.AuthInfos[context.AuthInfo]; ok {
			renameRequired = ComplexDiffRenameRequired
		}
		if renameRequired != ComplexDiffTypeNone {
			changeType |= ChangeType(ComplexDiffRenameRequired)
		}
		// New kubeconfig
		diff.Items = append(diff.Items, DiffItem{
			AffectedExisting: nil,
			AffectedIncoming: context,
			ChangeType:       changeType,
			Complex:          renameRequired,
		})
	}
	return diff, nil
}
