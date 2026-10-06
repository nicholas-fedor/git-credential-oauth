// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd"
)

// TestDocGenerator_Generate covers the entry point against the real tree.
//
// Every other generator test injects a mock, so this is the only test that
// proves the shipped command tree produces the published pages.
func TestDocGenerator_Generate(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		notWant []string
	}{
		{
			name: "documents every available command",
			want: []string{
				"_index.md",
				"capability/_index.md",
				"configure/_index.md",
				"get/_index.md",
				"unconfigure/_index.md",
				"version/_index.md",
			},
			notWant: []string{"store/_index.md", "erase/_index.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := NewDocGenerator()
			require.NoError(t, err)

			out := t.TempDir()

			require.NoError(t, generator.Generate(cmd.DocRoot(t.Context()), out))

			for _, page := range tt.want {
				assert.FileExists(t, filepath.Join(out, page))
			}

			for _, page := range tt.notWant {
				assert.NoFileExists(t, filepath.Join(out, page))
			}
		})
	}
}

// TestDocGenerator_GenerateWritesHugoFrontmatter checks the pages are usable
// by the site without hand editing.
func TestDocGenerator_GenerateWritesHugoFrontmatter(t *testing.T) {
	generator, err := NewDocGenerator()
	require.NoError(t, err)

	out := t.TempDir()

	require.NoError(t, generator.Generate(cmd.DocRoot(t.Context()), out))

	page, err := os.ReadFile(filepath.Join(out, "get", "_index.md"))
	require.NoError(t, err)

	assert.Contains(t, string(page), "type: docs")
	assert.Contains(t, string(page), "title: Get")
	assert.Contains(t, string(page), "git-credential-oauth get")
}

// TestDocGenerator_GenerateReportsAnUnusableOutputDirectory checks a failed
// setup is reported instead of writing a partial tree.
func TestDocGenerator_GenerateReportsAnUnusableOutputDirectory(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), filePerms))

	generator, err := NewDocGenerator()
	require.NoError(t, err)

	err = generator.Generate(cmd.DocRoot(t.Context()), filepath.Join(blocker, "cli"))
	require.Error(t, err)
}

func TestNewDocGeneratorWithDeps(t *testing.T) {
	tests := []struct {
		extractor DocExtractor
		renderer  TemplateRenderer
		name      string
		wantErr   bool
	}{
		{
			name:      "creates with valid deps",
			extractor: NewCobraExtractor(),
			renderer:  &mockRenderer{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDocGeneratorWithDeps(tt.extractor, tt.renderer)
			assert.NotNil(t, got)
		})
	}
}

func TestDocGenerator_generateAll(t *testing.T) {
	tests := []struct {
		doc     *commandDoc
		name    string
		wantErr bool
	}{
		{
			name:    "nil subcommands succeeds",
			doc:     &commandDoc{Name: "root", Index: &indexDoc{}},
			wantErr: false,
		},
		{
			name: "with subcommands",
			doc: &commandDoc{
				Name:  "root",
				Index: &indexDoc{},
				SubCommands: []*commandDoc{
					{Name: "sub1", Index: &indexDoc{}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDocGeneratorWithDeps(&mockExtractor{}, &mockRenderer{})
			tmpDir := t.TempDir()

			err := g.generateAll(tt.doc, tmpDir)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestDocGenerator_generateSection(t *testing.T) {
	tests := []struct {
		section *commandDoc
		name    string
		wantErr bool
	}{
		{
			name:    "simple section",
			section: &commandDoc{Name: "section", Index: &indexDoc{}},
			wantErr: false,
		},
		{
			name: "nested subcommands",
			section: &commandDoc{
				Name:  "section",
				Index: &indexDoc{},
				SubCommands: []*commandDoc{
					{Name: "sub", Index: &indexDoc{}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewDocGeneratorWithDeps(&mockExtractor{}, &mockRenderer{})
			tmpDir := t.TempDir()

			err := g.generateSection(tt.section, tmpDir)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

type mockRenderer struct{}

func (m *mockRenderer) RenderCommand(doc *commandDoc, path string) error {
	return nil
}

func (m *mockRenderer) RenderIndex(doc *indexDoc, path string) error {
	return nil
}

type mockExtractor struct{}

func (m *mockExtractor) Extract(cmd *cobra.Command, parentPath string) *commandDoc {
	return &commandDoc{
		Name:     cmd.Name(),
		Short:    cmd.Short,
		Long:     cmd.Long,
		Use:      cmd.Use,
		FullPath: parentPath + " " + cmd.Name(),
	}
}
