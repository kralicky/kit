package machinery

import "k8s.io/client-go/tools/clientcmd/api"

func (d *Diff) Apply(existing, incoming *api.Config, handler ConflictResolver) error {
	for _, item := range d.Items {
		// isComplex := (item.ChangeType & ChangeTypeComplex) != 0
		switch {
		case (item.ChangeType & ChangeTypeNew) != 0:
			clusterName := item.AffectedIncoming.Cluster
			authInfoName := item.AffectedIncoming.AuthInfo
			contextName := item.AffectedIncoming.Name
			if (item.Complex & ComplexDiffRenameRequired) != 0 {
				if _, exists := existing.Clusters[clusterName]; exists {
					clusterName = handler.Rename("Cluster", clusterName, func(s string) error {
						if _, exists := existing.Clusters[s]; exists {
							return ErrItemAlreadyExists
						}
						return nil
					})
				}
				if _, exists := existing.AuthInfos[authInfoName]; exists {
					authInfoName = handler.Rename("AuthInfo", authInfoName, func(s string) error {
						if _, exists := existing.AuthInfos[s]; exists {
							return ErrItemAlreadyExists
						}
						return nil
					})
				}
				if _, exists := existing.Contexts[contextName]; exists {
					contextName = handler.Rename("Context", contextName, func(s string) error {
						if _, exists := existing.Contexts[s]; exists {
							return ErrItemAlreadyExists
						}
						return nil
					})
				}
			}
			existing.Clusters[clusterName] = incoming.Clusters[item.AffectedIncoming.Cluster].DeepCopy()
			existing.AuthInfos[authInfoName] = incoming.AuthInfos[item.AffectedIncoming.AuthInfo].DeepCopy()
			existing.Contexts[contextName] = &api.Context{
				Cluster:  clusterName,
				AuthInfo: authInfoName,
			}
		case (item.ChangeType & ChangeTypeRename) != 0:
			existingContextName := item.AffectedExisting.Name
			existingClusterName := item.AffectedExisting.Cluster
			existingAuthInfoName := item.AffectedExisting.AuthInfo

			incomingContextName := item.AffectedIncoming.Name
			incomingClusterName := item.AffectedIncoming.Cluster
			incomingAuthInfoName := item.AffectedIncoming.AuthInfo

			if existingClusterName != incomingClusterName {
				existing.Clusters[incomingClusterName] = existing.Clusters[existingClusterName]
				delete(existing.Clusters, existingClusterName)
				existing.Contexts[existingContextName].Cluster = incomingClusterName
			}
			if existingAuthInfoName != incomingAuthInfoName {
				existing.AuthInfos[incomingAuthInfoName] = existing.AuthInfos[existingAuthInfoName]
				delete(existing.AuthInfos, existingAuthInfoName)
				existing.Contexts[existingContextName].AuthInfo = incomingAuthInfoName
			}
			if existingContextName != incomingContextName {
				existing.Contexts[incomingContextName] = existing.Contexts[existingContextName]
				delete(existing.Contexts, existingContextName)
			}
		case (item.ChangeType & ChangeTypeDelete) != 0:
			clusterName := item.AffectedExisting.Cluster
			authInfoName := item.AffectedExisting.AuthInfo
			contextName := item.AffectedExisting.Name
			delete(existing.Clusters, clusterName)
			delete(existing.AuthInfos, authInfoName)
			delete(existing.Contexts, contextName)
		case (item.ChangeType & ChangeTypeReplace) != 0:
			existingContextName := item.AffectedExisting.Name
			existingClusterName := item.AffectedExisting.Cluster
			existingAuthInfoName := item.AffectedExisting.AuthInfo

			delete(existing.Clusters, existingClusterName)
			delete(existing.AuthInfos, existingAuthInfoName)
			delete(existing.Contexts, existingContextName)

			existing.Clusters[item.AffectedIncoming.Cluster] =
				incoming.Clusters[item.AffectedIncoming.Cluster].DeepCopy()
			existing.AuthInfos[item.AffectedIncoming.AuthInfo] =
				incoming.AuthInfos[item.AffectedIncoming.AuthInfo].DeepCopy()
			existing.Contexts[item.AffectedIncoming.Name] =
				incoming.Contexts[item.AffectedIncoming.Name].DeepCopy()
		case (item.ChangeType & ChangeTypeModify) != 0:
			cmplx := item.Complex
			for cmplx != ComplexDiffTypeNone {
				switch {
				case (cmplx & ComplexDiffServerChanged) != 0:
					existing.Clusters[item.AffectedExisting.Cluster].Server =
						incoming.Clusters[item.AffectedIncoming.Cluster].Server
					cmplx &^= ComplexDiffServerChanged
				case (cmplx & ComplexDiffUserAuthChanged) != 0:
					existing.AuthInfos[item.AffectedExisting.AuthInfo] =
						incoming.AuthInfos[item.AffectedIncoming.AuthInfo].DeepCopy()
					cmplx &^= ComplexDiffUserAuthChanged
				case (cmplx & ComplexDiffClusterCAChanged) != 0:
					existing.Clusters[item.AffectedExisting.Cluster].CertificateAuthorityData =
						incoming.Clusters[item.AffectedIncoming.Cluster].CertificateAuthorityData
					cmplx &^= ComplexDiffClusterCAChanged
				case (cmplx & ComplexDiffPreferencesChanged) != 0:
					// No-op
					cmplx &^= ComplexDiffPreferencesChanged
				case (cmplx & ComplexDiffRenameRequired) != 0:
					// This isn't used in ChangeTypeModify
					panic("bug: ComplexDiffRenameRequired used in ChangeTypeModify")
				}
			}
		}
	}
	return nil
}
