// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:generate go run github.com/nicholas-fedor/git-credential-oauth/tools/docgen -out ../../docs/content/cli-reference

// Command docgen generates Hugo-compatible Markdown documentation for the CLI command tree.
//
// Usage:
//
//	go run ./tools/docgen -out ./docs/content/cli-reference
package main

import (
	"context"
	"flag"
	"log"

	"github.com/nicholas-fedor/git-credential-oauth/internal/cmd"
)

// main writes the CLI reference for the command tree.
//
// The tree is built with no dependencies because only command metadata is
// read; no command is executed.
func main() {
	out := flag.String("out", "./docs/content/cli-reference", "Output directory")

	flag.Parse()

	generator, err := NewDocGenerator()
	if err != nil {
		log.Fatalf("create doc generator: %v", err)
	}

	err = generator.Generate(cmd.DocRoot(context.Background()), *out)
	if err != nil {
		log.Fatal(err)
	}
}
