package machinery

import (
	"fmt"
	"math"
)

type ConflictResolver interface {
	Rename(kind string, oldName string, validator func(string) error) string
}

type autoResolver struct{}

func (r *autoResolver) Rename(
	kind string,
	oldName string,
	validator func(string) error,
) string {
	for i := 1; i < math.MaxInt; i++ {
		newName := fmt.Sprintf("%s-%d", oldName, i)
		if err := validator(newName); err == nil {
			return newName
		}
	}
	panic(fmt.Sprintf("failed to rename %s %s", kind, oldName))
}

var AutoResolver = &autoResolver{}
