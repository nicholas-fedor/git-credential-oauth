// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// DocExtractor defines the interface for extracting documentation from cobra commands.
type DocExtractor interface {
	// Extract builds a CommandDoc from a cobra command and its parent path.
	Extract(cmd *cobra.Command, parentPath string) *commandDoc
}

// cobraExtractor implements DocExtractor using cobra command metadata.
type cobraExtractor struct {
	// titleCaser converts command names to title case for display.
	titleCaser cases.Caser
}

const (
	// maxDescriptionLen is the maximum length for command descriptions before truncation.
	maxDescriptionLen = 160
)

// NewCobraExtractor creates a new cobraExtractor with English title casing.
//
// Returns:
//   - *cobraExtractor: A new extractor instance for cobra commands.
func NewCobraExtractor() *cobraExtractor {
	return &cobraExtractor{
		titleCaser: cases.Title(language.English),
	}
}

// Extract builds a CommandDoc from a cobra command and its parent path.
//
// Parameters:
//   - cmd: The cobra command to extract documentation from.
//   - parentPath: The full path of the parent command.
//
// Returns:
//   - *commandDoc: The extracted documentation data.
func (e *cobraExtractor) Extract(cmd *cobra.Command, parentPath string) *commandDoc {
	fullPath := e.buildFullPath(cmd, parentPath)

	doc := &commandDoc{
		Name:        cmd.Name(),
		Short:       cmd.Short,
		Long:        cmd.Long,
		Use:         cmd.Use,
		Example:     cmd.Example,
		UseLine:     e.buildUseLine(cmd.Use, fullPath),
		Title:       e.buildTitle(cmd),
		Description: e.buildDescription(cmd.Long),
		FullPath:    fullPath,
		Flags:       nil,
		Inherited:   nil,
		Examples:    nil,
		SubCommands: nil,
		HasSubs:     false,
		Index:       nil,
	}

	if cmd.Example != "" {
		doc.Examples = e.parseExamples(cmd.Example, fullPath, cmd.Use)
	}

	if cmd.HasAvailableFlags() {
		doc.Flags = e.buildFlags(cmd.Flags())
	}

	if cmd.HasAvailableInheritedFlags() {
		doc.Inherited = e.buildFlags(cmd.InheritedFlags())
	}

	if cmd.HasAvailableSubCommands() {
		doc.HasSubs = true
		doc.Index = e.buildIndex(cmd)
	}

	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}

		doc.SubCommands = append(doc.SubCommands, e.Extract(child, fullPath))
	}

	return doc
}

// buildDescription creates a truncated description from the long description.
//
// Newlines become spaces so the text fits a single Hugo frontmatter line.
//
// Parameters:
//   - long: The full long description text.
//
// Returns:
//   - string: The description, truncated to maxDescriptionLen if necessary.
func (e *cobraExtractor) buildDescription(long string) string {
	desc := strings.Join(strings.Fields(long), " ")

	if len(desc) > maxDescriptionLen {
		desc = desc[:maxDescriptionLen-3] + "..."
	}

	return desc
}

// buildFlags extracts documentation from a flag set.
//
// Parameters:
//   - flags: The flag set to extract documentation from.
//
// Returns:
//   - []flagDoc: A list of flag documentation entries.
func (e *cobraExtractor) buildFlags(flags *pflag.FlagSet) []flagDoc {
	var docs []flagDoc

	flags.VisitAll(func(flag *pflag.Flag) {
		docs = append(docs, flagDoc{
			Name:      flag.Name,
			Shorthand: flag.Shorthand,
			Default:   flag.DefValue,
			Type:      flag.Value.Type(),
			Usage:     flag.Usage,
		})
	})

	return docs
}

// buildFullPath constructs the full command path from parent and command name.
//
// Parameters:
//   - cmd: The cobra command.
//   - parentPath: The parent command path.
//
// Returns:
//   - string: The full command path (e.g., "mycli config edit").
func (e *cobraExtractor) buildFullPath(cmd *cobra.Command, parentPath string) string {
	if cmd.Name() == parentPath {
		return parentPath
	}

	return parentPath + " " + cmd.Name()
}

// buildIndex constructs the index documentation for a command group.
//
// Parameters:
//   - cmd: The parent cobra command with subcommands.
//
// Returns:
//   - *indexDoc: The index documentation with sections and entries.
func (e *cobraExtractor) buildIndex(cmd *cobra.Command) *indexDoc {
	index := &indexDoc{
		Title:       "CLI Reference",
		Description: "Complete command reference for git-credential-oauth.",
		Sections:    nil,
	}

	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}

		section := sectionEntry{
			Name:        child.Name(),
			Description: firstParagraph(child.Long),
			HasSubs:     child.HasAvailableSubCommands(),
			Title:       e.titleCase(child.Name()),
			SubCommands: nil,
		}

		if section.HasSubs {
			section.Title = child.Short
		}

		for _, sub := range child.Commands() {
			if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
				continue
			}

			url := fmt.Sprintf("/cli-reference/%s/%s/", child.Name(), sub.Name())
			if !section.HasSubs {
				url = fmt.Sprintf("/cli-reference/%s/", child.Name())
			}

			section.SubCommands = append(section.SubCommands, subCommandEntry{
				Name:        sub.Name(),
				Description: sub.Short,
				URL:         url,
			})
		}

		if !section.HasSubs {
			section.SubCommands = []subCommandEntry{
				{
					Name:        child.Name(),
					Description: child.Short,
					URL:         fmt.Sprintf("/cli-reference/%s/", child.Name()),
				},
			}
		}

		index.Sections = append(index.Sections, section)
	}

	return index
}

// buildTitle generates a display title for the command.
//
// Parameters:
//   - cmd: The cobra command.
//
// Returns:
//   - string: The title-cased command name or the Short description for command groups.
func (e *cobraExtractor) buildTitle(cmd *cobra.Command) string {
	if !cmd.HasAvailableSubCommands() {
		return e.titleCase(cmd.Name())
	}

	return cmd.Short
}

// buildUseLine constructs the usage line from the Use field and full path.
//
// Parameters:
//   - use: The Use field from the cobra command.
//   - fullPath: The full command path.
//
// Returns:
//   - string: The formatted usage line.
func (e *cobraExtractor) buildUseLine(use, fullPath string) string {
	if use == "" {
		return ""
	}

	args := strings.TrimPrefix(use, strings.Split(use, " ")[0])

	args = strings.TrimSpace(args)

	if args == "" {
		return fullPath
	}

	return fullPath + " " + args
}

// parseExamples parses example text into structured example documentation.
//
// Parameters:
//   - raw: The raw example text from the cobra command.
//   - fullPath: The full command path for context.
//   - use: The Use field from the cobra command.
//
// Returns:
//   - []exampleDoc: A list of parsed example documentation entries.
func (e *cobraExtractor) parseExamples(raw, fullPath, use string) []exampleDoc {
	var examples []exampleDoc

	usageCode := fullPath
	args := strings.TrimPrefix(use, strings.Split(use, " ")[0])

	args = strings.TrimSpace(args)
	if args != "" {
		usageCode = fullPath + " " + args
	}

	for para := range strings.SplitSeq(strings.TrimSpace(raw), "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		var (
			comment   string
			codeLines []string
		)

		for line := range strings.SplitSeq(para, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "#") {
				comment = trimmed
			} else if trimmed != "" {
				codeLines = append(codeLines, strings.TrimLeft(line, " "))
			}
		}

		if len(codeLines) == 0 {
			continue
		}

		code := strings.TrimSpace(strings.Join(codeLines, "\n"))

		title := strings.TrimSpace(strings.TrimPrefix(comment, "#"))

		title = strings.TrimSpace(title)

		examples = append(examples, exampleDoc{
			Title: title,
			Code:  code,
		})
	}

	if len(examples) == 1 && examples[0].Code == usageCode {
		return nil
	}

	return examples
}

// titleCase converts a string to title case using the extractor's caser.
//
// Parameters:
//   - s: The string to convert.
//
// Returns:
//   - string: The title-cased string.
func (e *cobraExtractor) titleCase(s string) string {
	return e.titleCaser.String(s)
}

// firstParagraph extracts the first paragraph from a multi-paragraph text.
//
// Newlines become spaces so the text fits a single table cell or frontmatter
// line.
//
// Parameters:
//   - text: The full text to extract from.
//
// Returns:
//   - string: The first paragraph with newlines replaced by spaces.
func firstParagraph(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "\n\n"); idx != -1 {
		text = text[:idx]
	}

	return strings.Join(strings.Fields(text), " ")
}
