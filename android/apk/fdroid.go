package repo

import (
	"fmt"
)

type FDroidPackage string

func (pkg FDroidPackage) Name() string {
	return string(pkg)
}

func (pkg FDroidPackage) Source() string {
	return "F-Droid"
}

func (pkg FDroidPackage) UpdateCache(_ string) (string, error) {
	return "", fmt.Errorf("Not implemented")
}
