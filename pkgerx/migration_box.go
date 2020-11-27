package pkgerx

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gobuffalo/pop/v5"
	"github.com/markbates/pkger"
	"github.com/pkg/errors"

	"github.com/ory/x/logrusx"
)

type (
	// MigrationBox is a wrapper around pkger.Dir and Migrator.
	// This will allow you to run migrations from migrations packed
	// inside of a compiled binary.
	MigrationBox struct {
		pop.Migrator
		Dir pkger.Dir
		l   *logrusx.Logger
	}
)

// NewMigrationBox from a packr.Dir and a Connection.
//
//	migrations, err := NewMigrationBox(pkger.Dir("/migrations"))
//
func NewMigrationBox(dir pkger.Dir, c *pop.Connection, l *logrusx.Logger) (*MigrationBox, error) {
	mb := MigrationBox{
		Migrator: pop.NewMigrator(c),
		Dir:      dir,
		l:        l,
	}

	runner := func(f io.Reader) func(mf pop.Migration, tx *pop.Connection) error {
		return func(mf pop.Migration, tx *pop.Connection) error {
			content, err := pop.MigrationContent(mf, tx, f, true)
			if err != nil {
				return errors.Wrapf(err, "error processing %s", mf.Path)
			}
			if content == "" {
				return nil
			}
			err = tx.RawQuery(content).Exec()
			if err != nil {
				return errors.Wrapf(err, "error executing %s, sql: %s", mf.Path, content)
			}
			return nil
		}
	}

	err := mb.findMigrations(runner)
	if err != nil {
		return &mb, err
	}

	return &mb, nil
}

func (fm *MigrationBox) findMigrations(runner func(f io.Reader) func(mf pop.Migration, tx *pop.Connection) error) error {
	return pkger.Walk(string(fm.Dir), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		match, err := pop.ParseMigrationFilename(info.Name())
		if err != nil {
			if strings.HasPrefix(err.Error(), "unsupported dialect") {
				fm.l.Debugf("Ignoring migration file %s because dialect is not supported: %s", info.Name(), err.Error())
				return nil
			}
			return err
		}

		if match == nil {
			fm.l.Debugf("Ignoring migration file %s because it does not match the file pattern.", info.Name())
			return nil
		}

		file, err := pkger.Open(p)
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		mf := pop.Migration{
			Path:      p,
			Version:   match.Version,
			Name:      match.Name,
			DBType:    match.DBType,
			Direction: match.Direction,
			Type:      match.Type,
			Runner:    runner(bytes.NewReader(content)),
		}
		fm.Migrations[mf.Direction] = append(fm.Migrations[mf.Direction], mf)
		return nil
	})
}
