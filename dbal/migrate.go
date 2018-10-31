package dbal

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

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

func NewMustPackerMigrationSource(l logrus.FieldLogger, folder []string) *migrate.PackrMigrationSource {
	m, err := NewPackerMigrationSource(l, folder)
	if err != nil {
		l.WithError(err).Fatal("Unable to set up migration source")
	}
	return m
}

func NewPackerMigrationSource(l logrus.FieldLogger, folder []string) (*migrate.PackrMigrationSource, error) {
	b := packr.NewBox(migrationBasePath)
	var files migrationFiles

	for _, f := range folder {
		if err := filepath.Walk(f, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			if filepath.Ext(path) != ".sql" {
				return nil
			}

			abs, err := filepath.Abs(path)
			if err != nil {
				return errors.WithStack(err)
			}

			l.WithField("file", abs).Debugf("Processing sql migration file")

			body, err := ioutil.ReadFile(abs)
			if err != nil {
				return errors.WithStack(err)
			}

			files = append(files, migrationFile{
				Filename: filepath.Base(path),
				Filepath: abs,
				Content:  body,
			})

			return nil
		}); err != nil {
			return nil, err
		}
	}

	sort.Sort(files)

	for _, f := range files {
		b.AddBytes(filepath.ToSlash(filepath.Join(migrationBasePath, f.Filename)), f.Content)
	}

	return &migrate.PackrMigrationSource{
		Box: b,
		Dir: migrationBasePath,
	}, nil
}
