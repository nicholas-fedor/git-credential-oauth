// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

// commandDoc represents a command's documentation data for template rendering.
type commandDoc struct {
	Index       *indexDoc
	Title       string
	FullPath    string
	Use         string
	Example     string
	UseLine     string
	Name        string
	Description string
	Long        string
	Short       string
	Inherited   []flagDoc
	Examples    []exampleDoc
	SubCommands []*commandDoc
	Flags       []flagDoc
	HasSubs     bool
}

// flagDoc represents a flag's documentation data.
//
// Fields stay exported because text/template can only read exported fields.
type flagDoc struct {
	// Name is the flag name.
	Name string
	// Shorthand is the single-character shorthand.
	Shorthand string
	// Default is the default value as a string.
	Default string
	// Type is the flag value type.
	Type string
	// Usage is the usage description.
	Usage string
}

// exampleDoc represents an example block's documentation data.
type exampleDoc struct {
	// Title is the example title.
	Title string
	// Code is the example code.
	Code string
}

// indexDoc represents the index page's documentation data.
type indexDoc struct {
	// Title is the index page title.
	Title string
	// Description is the index page description.
	Description string
	// Sections contains the index sections.
	Sections []sectionEntry
}

// sectionEntry represents a section in the index page.
type sectionEntry struct {
	Name        string
	Title       string
	Description string
	SubCommands []subCommandEntry
	HasSubs     bool
}

// subCommandEntry represents a subcommand entry in a section.
type subCommandEntry struct {
	// Name is the subcommand name.
	Name string
	// Description is the subcommand description.
	Description string
	// URL is the documentation URL path.
	URL string
}
