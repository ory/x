package dbal

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	"github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
)

type migrationFile struct {
	Filename string
	Filepath string
	Content  []byte
}

const migrationBasePath = "/migrations/sql"

type migrationFiles []migrationFile

func (s migrationFiles) Len() int           { return len(s) }
func (s migrationFiles) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s migrationFiles) Less(i, j int) bool { return s[i].Filename < s[j].Filename }

// NewMustPackerMigrationSource create a new packr-based migration source or fatals.
func NewMustPackerMigrationSource(l logrus.FieldLogger, folder []string, loader func(string) ([]byte, error), filters []string) *migrate.PackrMigrationSource {
	m, err := NewPackerMigrationSource(l, folder, loader, filters)
	if err != nil {
		l.WithError(err).WithField("stack", fmt.Sprintf("%+v", err)).Fatal("Unable to set up migration source")
	}
	return m
}

// NewPackerMigrationSource create a new packr-based migration source or returns an error
func NewPackerMigrationSource(l logrus.FieldLogger, sources []string, loader func(string) ([]byte, error), filters []string) (*migrate.PackrMigrationSource, error) {
	b := packr.NewBox(migrationBasePath)
	var files migrationFiles

	for _, source := range sources {
		if filepath.Ext(source) != ".sql" {
			continue
		}

		var found bool
		for _, f := range filters {
			if strings.Contains(source, f) {
				found = true
			}
		}

		if !found {
			continue
		}

		l.WithField("file", source).Debugf("Processing sql migration file")

		body, err := loader(source)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		files = append(files, migrationFile{
			Filename: filepath.Base(source),
			Filepath: source,
			Content:  body,
		})
	}

	sort.Sort(files)

	for _, f := range files {
		b.AddBytes(filepath.ToSlash(filepath.Join(migrationBasePath, f.Filename)), f.Content)
		//if err := b.AddBytes(filepath.ToSlash(filepath.Join(migrationBasePath, f.Filename)), f.Content); err != nil {
		//	return nil, errors.WithStack(err)
		//}
	}

	return &migrate.PackrMigrationSource{
		Box: b,
		Dir: migrationBasePath,
	}, nil
}
