// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package pkgerx

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/gobuffalo/fizz"
	"github.com/gobuffalo/pop/v6"
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

		Dir              pkger.Dir
		l                *logrusx.Logger
		migrationContent MigrationContent
	}
	MigrationContent func(mf pop.Migration, c *pop.Connection, r io.Reader, usingTemplate bool) (string, error)
)

func templatingMigrationContent(params map[string]interface{}) func(pop.Migration, *pop.Connection, io.Reader, bool) (string, error) {
	return func(mf pop.Migration, c *pop.Connection, r io.Reader, usingTemplate bool) (string, error) {
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return "", nil
		}

		content := ""
		if usingTemplate {
			t := template.New("migration")
			t.Funcs(SQLTemplateFuncs)
			t, err := t.Parse(string(b))
			if err != nil {
				return "", err
			}

			var bb bytes.Buffer
			err = t.Execute(&bb, struct {
				DialectDetails *pop.ConnectionDetails
				Parameters     map[string]interface{}
			}{
				DialectDetails: c.Dialect.Details(),
				Parameters:     params,
			})
			if err != nil {
				return "", errors.Wrapf(err, "could not execute migration template %s", mf.Path)
			}
			content = bb.String()
		} else {
			content = string(b)
		}

		if mf.Type == "fizz" {
			content, err = fizz.AString(content, c.Dialect.FizzTranslator())
			if err != nil {
				return "", errors.Wrapf(err, "could not fizz the migration %s", mf.Path)
			}
		}

		return content, nil
	}
}

func WithTemplateValues(v map[string]interface{}) func(*MigrationBox) *MigrationBox {
	return func(m *MigrationBox) *MigrationBox {
		m.migrationContent = templatingMigrationContent(v)
		return m
	}
}

func WithMigrationContentMiddleware(middleware func(content string, err error) (string, error)) func(*MigrationBox) *MigrationBox {
	return func(m *MigrationBox) *MigrationBox {
		prev := m.migrationContent
		m.migrationContent = func(mf pop.Migration, c *pop.Connection, r io.Reader, usingTemplate bool) (string, error) {
			return middleware(prev(mf, c, r, usingTemplate))
		}
		return m
	}
}

// NewMigrationBox from a packr.Dir and a Connection.
//
//	migrations, err := NewMigrationBox(pkger.Dir("/migrations"))
func NewMigrationBox(dir pkger.Dir, c *pop.Connection, l *logrusx.Logger, opts ...func(*MigrationBox) *MigrationBox) (*MigrationBox, error) {
	mb := &MigrationBox{
		Migrator:         pop.NewMigrator(c),
		Dir:              dir,
		l:                l,
		migrationContent: pop.MigrationContent,
	}

	for _, o := range opts {
		mb = o(mb)
	}

	runner := func(f io.Reader) func(mf pop.Migration, tx *pop.Connection) error {
		return func(mf pop.Migration, tx *pop.Connection) error {
			content, err := mb.migrationContent(mf, tx, f, true)
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
		return mb, err
	}

	return mb, nil
}

func (fm *MigrationBox) findMigrations(runner func(f io.Reader) func(mf pop.Migration, tx *pop.Connection) error) error {
	return pkger.Walk(string(fm.Dir), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
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

		mf := pop.Migration{
			Path:      p,
			Version:   match.Version,
			Name:      match.Name,
			DBType:    match.DBType,
			Direction: match.Direction,
			Type:      match.Type,
			Runner:    runner(bytes.NewReader(content)),
		}

		switch mf.Direction {
		case "down":
			fm.DownMigrations.Migrations = append(fm.DownMigrations.Migrations, mf)
		case "up":
			fm.UpMigrations.Migrations = append(fm.UpMigrations.Migrations, mf)
		default:
			return errors.Errorf("unknown direction %s", mf.Direction)
		}

		return nil
	})
}
