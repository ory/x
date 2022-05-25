package popx

import (
	"io"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/gobuffalo/pop/v6"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
)

type (
	// MigrationBox is a embed migration box.
	MigrationBox struct {
		*Migrator

		Dir              fs.FS
		l                *logrusx.Logger
		migrationContent MigrationContent
		goMigrations     Migrations
	}
	MigrationContent func(mf Migration, c *pop.Connection, r []byte, usingTemplate bool) (string, error)
	GoMigration      func(c *pop.Tx) error
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

// WithGoMigrations adds migrations that have a custom migration runner.
// TEST THEM THOROUGHLY!
// It will be very hard to fix a buggy migration.
func WithGoMigrations(migrations Migrations) func(*MigrationBox) *MigrationBox {
	return func(m *MigrationBox) *MigrationBox {
		m.goMigrations = migrations
		return m
	}
}

// WithTestdata
func WithTestdata(t *testing.T, testdata fs.FS) func(*MigrationBox) *MigrationBox {
	testdataPattern := regexp.MustCompile(`^(\d+)_testdata(|\.[a-zA-Z0-9]+).sql$`)
	return func(m *MigrationBox) *MigrationBox {
		require.NoError(t, fs.WalkDir(testdata, ".", func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			match := testdataPattern.FindStringSubmatch(info.Name())
			if len(match) != 2 && len(match) != 3 {
				t.Logf(`WARNING! Found a test migration which does not match the test data pattern: %s`, info.Name())
				return nil
			}

			version := match[1]
			flavor := "all"
			if len(match) == 3 && len(match[2]) > 0 {
				flavor = pop.NormalizeSynonyms(strings.TrimPrefix(match[2], "."))
			}

			//t.Logf("Found test migration \"%s\" (%s, %+v): %s", flavor, match, err, info.Name())

			m.Migrations["up"] = append(m.Migrations["up"], Migration{
				Version:   version + "9", // run testdata after version
				Path:      path,
				Name:      info.Name(),
				DBType:    flavor,
				Direction: "up",
				Type:      "sql",
				Runner: func(m Migration, _ *pop.Connection, tx *pop.Tx) error {
					match := match
					b, err := fs.ReadFile(testdata, m.Path)
					if err != nil {
						return err
					}
					_, err = tx.Exec(string(b))
					//t.Logf("Ran test migration \"%s\" (%s, %+v) with error \"%v\" and content:\n %s", m.Path, m.DBType, match, err, string(b))
					return err
				},
			})

			m.Migrations["down"] = append(m.Migrations["down"], Migration{
				Version:   version + "9", // run testdata after version
				Path:      path,
				Name:      info.Name(),
				DBType:    flavor,
				Direction: "down",
				Type:      "sql",
				Runner: func(m Migration, _ *pop.Connection, tx *pop.Tx) error {
					return nil
				},
			})

			sort.Sort(sortIdent(m.Migrations["up"]))
			sort.Sort(sort.Reverse(sortIdent(m.Migrations["down"])))
			return nil
		}))
		return m
	}
}

// NewMigrationBox creates a new migration box.
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

	for _, migration := range mb.goMigrations {
		mb.Migrations[migration.Direction] = append(mb.Migrations[migration.Direction], migration)
	}

	if err := mb.check(); err != nil {
		return nil, err
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

// hasDownMigrationWithVersion checks if there is a migration with the given
// version.
func (fm *MigrationBox) hasDownMigrationWithVersion(version string) bool {
	for _, down := range fm.Migrations["down"] {
		if version == down.Version {
			return true
		}
	}
	return false
}

// check checks that every "up" migration has a corresponding "down" migration.
func (fm *MigrationBox) check() error {
	for _, up := range fm.Migrations["up"] {
		if !fm.hasDownMigrationWithVersion(up.Version) {
			return errors.Errorf("migration %s has no corresponding down migration", up.Version)
		}
	}
	return nil
}
