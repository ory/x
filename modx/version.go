package modx

import (
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

func FindVersion(gomod []byte, module string) (string, error) {
	m, err := modfile.Parse("go.mod", gomod, nil)
	if err != nil {
		return "", err
	}

	for _, r := range m.Require {
		if r.Mod.Path == module {
			return r.Mod.Version, nil
		}
	}

	return "", errors.Errorf("no go.mod entry found for: %s", module)
}
