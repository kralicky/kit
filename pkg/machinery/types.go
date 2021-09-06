package machinery

import "k8s.io/client-go/tools/clientcmd/api"

type NamedContext struct {
	*api.Context
	Name string
}

func NewNamedContext(name string, context *api.Context) NamedContext {
	return NamedContext{
		Name:    name,
		Context: context.DeepCopy(),
	}
}

func NamedContextFrom(contexts map[string]*api.Context, name string) NamedContext {
	return NewNamedContext(name, contexts[name])
}

type ChangeType int

const (
	// A new kubeconfig was added and does not interfere with any existing ones
	ChangeTypeNew ChangeType = 1 << iota

	// A kubeconfig was renamed but otherwise has no conflicts
	ChangeTypeRename

	// An existing kubeconfig was removed
	ChangeTypeDelete

	// A new kubeconfig completely replaces an existing one
	ChangeTypeReplace

	// A new kubeconfig completely replaces an existing one
	ChangeTypeModify

	// Some additional things need to be taken care of
	ChangeTypeComplex
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
	AffectedIncoming NamedContext
	AffectedExisting NamedContext
	ChangeType       ChangeType
	Complex          ComplexDiffType
}
