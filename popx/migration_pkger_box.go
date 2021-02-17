package popx

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/gobuffalo/pop/v5"
	"github.com/markbates/pkger"
	"github.com/pkg/errors"

	"github.com/ory/x/logrusx"
)

type (
	// MigrationBoxPkger is a wrapper around pkger.Dir and Migrator.
	// This will allow you to run migrations from migrations packed
	// inside of a compiled binary.
	MigrationBoxPkger struct {
		Migrator

		Dir              pkger.Dir
		l                *logrusx.Logger
		migrationContent MigrationContent
	}
	MigrationContent func(mf Migration, c *pop.Connection, r []byte, usingTemplate bool) (string, error)
)

func WithTemplateValues(v map[string]interface{}) func(*MigrationBoxPkger) *MigrationBoxPkger {
	return func(m *MigrationBoxPkger) *MigrationBoxPkger {
		m.migrationContent = ParameterizedMigrationContent(v)
		return m
	}
}

func WithMigrationContentMiddleware(middleware func(content string, err error) (string, error)) func(*MigrationBoxPkger) *MigrationBoxPkger {
	return func(m *MigrationBoxPkger) *MigrationBoxPkger {
		prev := m.migrationContent
		m.migrationContent = func(mf Migration, c *pop.Connection, r []byte, usingTemplate bool) (string, error) {
			return middleware(prev(mf, c, r, usingTemplate))
		}
		return m
	}
}

// NewMigrationBoxPkger from a packr.Dir and a Connection.
//
//	migrations, err := NewMigrationBoxPkger(pkger.Dir("/migrations"))
//
func NewMigrationBoxPkger(dir pkger.Dir, c *pop.Connection, l *logrusx.Logger, opts ...func(*MigrationBoxPkger) *MigrationBoxPkger) (*MigrationBoxPkger, error) {
	mb := &MigrationBoxPkger{
		Migrator:         NewMigrator(c, l),
		Dir:              dir,
		l:                l,
		migrationContent: ParameterizedMigrationContent(nil),
	}

	for _, o := range opts {
		mb = o(mb)
	}

	runner := func(b []byte) func(Migration, *pop.Connection, *pop.Tx) error {
		return func(mf Migration, c *pop.Connection, tx *pop.Tx) error {
			content, err := mb.migrationContent(mf, c, b, true)
			if err != nil {
				return errors.Wrapf(err, "error processing %s", mf.Path)
			}
			if content == "" {
				l.WithField("migration", mf.Path).Warn("Ignoring migration because content is empty.")
				return nil
			}
			if _, err = tx.Exec(content); err != nil {
				return errors.Wrapf(err, "error executing %s, sql: %s", mf.Path, content)
			}
			return nil
		}
	}

	err := mb.findMigrations(runner)
	if err != nil {
		return mb, err
	}

	return mb, nil
}

func (fm *MigrationBoxPkger) findMigrations(runner func([]byte) func(mf Migration, c *pop.Connection, tx *pop.Tx) error) error {
	return pkger.Walk(string(fm.Dir), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		if info.IsDir() {
			return nil
		}

		match, err := pop.ParseMigrationFilename(info.Name())
		if err != nil {
			if strings.HasPrefix(err.Error(), "unsupported dialect") {
				fm.l.Debugf("Ignoring migration file %s because dialect is not supported: %s", info.Name(), err.Error())
				return nil
			}
			return errors.WithStack(err)
		}

		if match == nil {
			fm.l.Debugf("Ignoring migration file %s because it does not match the file pattern.", info.Name())
			return nil
		}

		file, err := pkger.Open(p)
		if err != nil {
			return errors.WithStack(err)
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return errors.WithStack(err)
		}

		mf := Migration{
			Path:      p,
			Version:   match.Version,
			Name:      match.Name,
			DBType:    match.DBType,
			Direction: match.Direction,
			Type:      match.Type,
			Runner:    runner(content),
		}
		fm.Migrations[mf.Direction] = append(fm.Migrations[mf.Direction], mf)
		return nil
	})
}
