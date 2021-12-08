package popx

import (
	"io"
	"io/fs"
	"sort"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/pkg/errors"

	"github.com/ory/x/logrusx"
)

type (
	// MigrationBox is a embed migration box.
	MigrationBox struct {
		*Migrator

		Dir              fs.FS
		l                *logrusx.Logger
		migrationContent MigrationContent
	}
	MigrationContent func(mf Migration, c *pop.Connection, r []byte, usingTemplate bool) (string, error)
)

func WithTemplateValues(v map[string]interface{}) func(*MigrationBox) *MigrationBox {
	return func(m *MigrationBox) *MigrationBox {
		m.migrationContent = ParameterizedMigrationContent(v)
		return m
	}
}

func WithMigrationContentMiddleware(middleware func(content string, err error) (string, error)) func(*MigrationBox) *MigrationBox {
	return func(m *MigrationBox) *MigrationBox {
		prev := m.migrationContent
		m.migrationContent = func(mf Migration, c *pop.Connection, r []byte, usingTemplate bool) (string, error) {
			return middleware(prev(mf, c, r, usingTemplate))
		}
		return m
	}
}

// NewMigrationBox from a packr.Dir and a Connection.
//
//	migrations, err := NewMigrationBox(pkger.Dir("/migrations"))
//
func NewMigrationBox(dir fs.FS, m *Migrator, opts ...func(*MigrationBox) *MigrationBox) (*MigrationBox, error) {
	mb := &MigrationBox{
		Migrator:         m,
		Dir:              dir,
		l:                m.l,
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
				m.l.WithField("migration", mf.Path).Trace("This is usually ok - ignoring migration because content is empty. This is ok!")
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

func (fm *MigrationBox) findMigrations(runner func([]byte) func(mf Migration, c *pop.Connection, tx *pop.Tx) error) error {
	return fs.WalkDir(fm.Dir, ".", func(p string, info fs.DirEntry, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		if info.IsDir() {
			return nil
		}

		match, err := pop.ParseMigrationFilename(info.Name())
		if err != nil {
			if strings.HasPrefix(err.Error(), "unsupported dialect") {
				fm.l.Tracef("This is usually ok - ignoring migration file %s because dialect is not supported: %s", info.Name(), err.Error())
				return nil
			}
			return errors.WithStack(err)
		}

		if match == nil {
			fm.l.Tracef("This is usually ok - ignoring migration file %s because it does not match the file pattern.", info.Name())
			return nil
		}

		f, err := fm.Dir.Open(p)
		if err != nil {
			return errors.WithStack(err)
		}
		content, err := io.ReadAll(f)
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
		mod := sortIdent(fm.Migrations[mf.Direction])
		if mf.Direction == "down" {
			mod = sort.Reverse(mod)
		}
		sort.Sort(mod)
		return nil
	})
}
