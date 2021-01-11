package clidoc

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Generate generates markdown documentation for a cobra command and its children.
func Generate(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("command expects one argument which is the project's root folder")
	}

	navItems := make([]string, 0)
	if err := generate(cmd, args[0], &navItems); err != nil {
		return err
	}
	sort.Strings(navItems)

	spath := filepath.Join(args[0], "docs", "sidebar.json")
	sidebar, err := ioutil.ReadFile(spath)
	if err != nil {
		return err
	}

	if !gjson.ValidBytes(sidebar) {
		return errors.New("sidebar file is not valid JSON")
	}

	var index int
	gjson.GetBytes(sidebar, `Reference`).ForEach(func(key, value gjson.Result) bool {
		if strings.Contains(value.Get("label").String(), "CLI") {
			return false
		}
		index++
		return true
	})

	sidebar, err = sjson.SetBytes(sidebar, fmt.Sprintf(`Reference.%d.items`, index), navItems)
	if err != nil {
		return err
	}

	/* #nosec G306 - TODO evaluate why */
	if err := ioutil.WriteFile(spath, []byte(gjson.GetBytes(sidebar, "@pretty").Raw), 0644); err != nil {
		return err
	}

	return nil
}

func trimExt(s string) string {
	return strings.ReplaceAll(strings.TrimSuffix(s, filepath.Ext(s)), "_", "-")
}

func generate(cmd *cobra.Command, dir string, navItems *[]string) error {
	cmd.DisableAutoGenTag = true
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := generate(c, dir, navItems); err != nil {
			return err
		}
	}

	basename := strings.Replace(cmd.CommandPath(), " ", "-", -1)
	if err := os.MkdirAll(filepath.Join(dir, "docs", "docs", "cli"), 0755); err != nil {
		return err
	}

	filename := filepath.Join(dir, "docs", "docs", "cli", basename) + ".md"
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, fmt.Sprintf(`---
id: %s
title: %s
description: %s %s
---

<!--
This file is auto-generated.

To improve this file please make your change against the appropriate "./cmd/*.go" file.
-->
`,
		basename,
		cmd.CommandPath(),
		cmd.CommandPath(),
		cmd.Short,
	)); err != nil {
		return err
	}

	*navItems = append(*navItems, filepath.Join("cli", basename))
	if err := doc.GenMarkdownCustom(cmd, f, trimExt); err != nil {
		return err
	}

	return nil
}
