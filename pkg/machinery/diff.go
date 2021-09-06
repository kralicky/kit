package machinery

import (
	"bytes"
	"reflect"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/tools/clientcmd/api"
)

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
	// First pass handles new, renamed, or replaced contexts:
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
		namedContext := NewNamedContext(contextName, context)
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
		if existingContext, ok := existing.Contexts[contextName]; ok {
			existingCluster, ok := existing.Clusters[existingContext.Cluster]
			if ok && existingContext.Cluster == context.Cluster {
				existingAuth, ok := existing.AuthInfos[existingContext.AuthInfo]
				if ok && existingContext.AuthInfo == context.AuthInfo {
					if ClustersEqual(existingCluster, incomingCluster) &&
						AuthInfosEqual(existingAuth, incomingAuth) {
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
			Context NamedContext
		}
		var matchingAuth *struct {
			AuthInfo *api.AuthInfo
			Context  NamedContext
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
					Context NamedContext
				}{
					Cluster: existingCluster,
					Context: NewNamedContext(existingContextName, existingContext),
				}
			}
			if AuthInfosEqual(existingAuth, incomingAuth) {
				matchingAuth = &struct {
					AuthInfo *api.AuthInfo
					Context  NamedContext
				}{
					AuthInfo: existingAuth,
					Context:  NewNamedContext(existingContextName, existingContext),
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
			// Renamed
			diff.Items = append(diff.Items, DiffItem{
				AffectedExisting: matchingCluster.Context,
				AffectedIncoming: namedContext,
				ChangeType:       ChangeTypeRename,
				Complex:          ComplexDiffTypeNone,
			})
			continue
		case matchingCluster != nil:
			// User auth changed
			diff.Items = append(diff.Items, DiffItem{
				AffectedExisting: matchingCluster.Context,
				AffectedIncoming: namedContext,
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
				AffectedIncoming: namedContext,
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
						AffectedExisting: NewNamedContext(existingContextName, existingContext),
						AffectedIncoming: namedContext,
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
					AffectedExisting: NewNamedContext(existingContextName, existingContext),
					AffectedIncoming: namedContext,
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
			AffectedExisting: NamedContext{},
			AffectedIncoming: namedContext,
			ChangeType:       changeType,
			Complex:          renameRequired,
		})
	}

	// Second pass handles deleted contexts
	// 1. For each existing context, check if there is a corresponding diff
	// 	  entry with the context marked as the affected existing context.
	//    If so, skip it.
	// 2. If there is no corresponding diff entry, check if there is an exact
	//    match in the incoming contexts (if so, it would have been skipped
	//		in the first pass). If so, skip it.
	// 3. If there is no exact match, mark it as deleted.
	for existingContextName, existingContext := range existing.Contexts {
		var found bool
		for _, diffItem := range diff.Items {
			if diffItem.AffectedExisting.Name == existingContextName {
				found = true
				break
			}
		}
		if found {
			continue
		}
		for incomingContextName, incomingContext := range incoming.Contexts {
			existingCluster, ok := existing.Clusters[existingContext.Cluster]
			if !ok {
				log.Fatalf("Local config is ill-formed: context %s references nonexistent cluster %s", existingContextName, existingContext.Cluster)
			}
			existingAuth, ok := existing.AuthInfos[existingContext.AuthInfo]
			if !ok {
				log.Fatalf("Local config is ill-formed: context %s references nonexistent auth info %s", existingContextName, existingContext.AuthInfo)
			}
			incomingCluster, ok := incoming.Clusters[incomingContext.Cluster]
			if !ok {
				log.Fatalf("Incoming config is ill-formed: context %s references nonexistent cluster %s", incomingContextName, incomingContext.Cluster)
			}
			incomingAuth, ok := incoming.AuthInfos[incomingContext.AuthInfo]
			if !ok {
				log.Fatalf("Incoming config is ill-formed: context %s references nonexistent auth info %s", incomingContextName, incomingContext.AuthInfo)
			}
			if ClustersEqual(existingCluster, incomingCluster) &&
				AuthInfosEqual(existingAuth, incomingAuth) {
				found = true
				break
			}
		}
		if found {
			continue
		}

		diff.Items = append(diff.Items, DiffItem{
			AffectedExisting: NewNamedContext(existingContextName, existingContext),
			AffectedIncoming: NamedContext{},
			ChangeType:       ChangeTypeDelete,
			Complex:          ComplexDiffTypeNone,
		})
	}
	return diff, nil
}
