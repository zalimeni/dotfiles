// Package cleanup provides post-processing functions for pandoc-generated
// markdown. It normalizes heading levels, strips excessive blank lines,
// converts absolute image paths to relative, and removes Word-specific
// artifacts such as []{.underline} spans.
package cleanup

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Clean applies all post-processing transformations to markdown content.
// The imagesDir parameter is used to convert absolute image paths to relative;
// pass an empty string to skip image path conversion.
func Clean(md string, imagesDir string) string {
	md = normalizeHeadings(md)
	md = stripWordArtifacts(md)
	md = convertImagePaths(md, imagesDir)
	md = collapseBlankLines(md)
	return md
}

// headingRe matches ATX-style markdown headings (e.g., "### Heading").
var headingRe = regexp.MustCompile(`(?m)^(#{1,6})\s`)

// normalizeHeadings shifts heading levels so the minimum heading in the
// document becomes H1. For example, if the smallest heading is H3, all
// headings are shifted up by 2 levels.
func normalizeHeadings(md string) string {
	matches := headingRe.FindAllStringSubmatch(md, -1)
	if len(matches) == 0 {
		return md
	}

	minLevel := 7
	for _, m := range matches {
		level := len(m[1])
		if level < minLevel {
			minLevel = level
		}
	}

	if minLevel <= 1 {
		return md
	}

	shift := minLevel - 1
	return headingRe.ReplaceAllStringFunc(md, func(match string) string {
		// Count leading '#' characters.
		hashes := 0
		for _, ch := range match {
			if ch == '#' {
				hashes++
			} else {
				break
			}
		}
		newLevel := hashes - shift
		if newLevel < 1 {
			newLevel = 1
		}
		return strings.Repeat("#", newLevel) + match[hashes:]
	})
}

// blankLineRe matches runs of 3 or more consecutive newlines (i.e., 2+ blank lines).
var blankLineRe = regexp.MustCompile(`\n{3,}`)

// collapseBlankLines reduces runs of multiple blank lines to a single blank
// line (two newlines).
func collapseBlankLines(md string) string {
	return blankLineRe.ReplaceAllString(md, "\n\n")
}

// wordArtifactRe matches Word-specific span artifacts like []{.underline} or
// [text]{.underline}. It captures the inner text so the replacement preserves
// content while stripping the span wrapper.
var wordArtifactRe = regexp.MustCompile(`\[([^\]]*)\]\{[^}]*\}`)

// stripWordArtifacts removes Word-specific span artifacts from the markdown,
// preserving the inner text content.
func stripWordArtifacts(md string) string {
	return wordArtifactRe.ReplaceAllString(md, "$1")
}

// imagePathRe matches markdown image references: ![alt](path)
var imagePathRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// convertImagePaths converts absolute image paths to relative paths. If
// imagesDir is empty, no conversion is performed.
func convertImagePaths(md string, imagesDir string) string {
	if imagesDir == "" {
		return md
	}

	absDir, err := filepath.Abs(imagesDir)
	if err != nil {
		return md
	}

	return imagePathRe.ReplaceAllStringFunc(md, func(match string) string {
		sub := imagePathRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		alt := sub[1]
		imgPath := sub[2]

		if !filepath.IsAbs(imgPath) {
			return match
		}

		rel, err := filepath.Rel(filepath.Dir(absDir), imgPath)
		if err != nil {
			return match
		}

		return "![" + alt + "](" + rel + ")"
	})
}
