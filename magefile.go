//go:build mage

package main

import (
	"github.com/magefile/mage/sh"
)

var Default = Build

func Build() error {
	return sh.Run("go", "build", "-o", "bin/kit", "./cmd/kit")
}
