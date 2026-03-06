package cleanup

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseBlankLines_MultipleBlankLines(t *testing.T) {
	input := "line1\n\n\n\nline2\n\n\n\n\nline3"
	got := collapseBlankLines(input)
	assert.Equal(t, "line1\n\nline2\n\nline3", got)
}

func TestCollapseBlankLines_SingleBlankLine(t *testing.T) {
	input := "line1\n\nline2"
	got := collapseBlankLines(input)
	assert.Equal(t, "line1\n\nline2", got)
}

func TestCollapseBlankLines_NoBlankLines(t *testing.T) {
	input := "line1\nline2\nline3"
	got := collapseBlankLines(input)
	assert.Equal(t, input, got)
}

func TestNormalizeHeadings_ShiftH3ToH1(t *testing.T) {
	input := "### Title\n\nSome text\n\n#### Subtitle\n\n##### Deep\n"
	got := normalizeHeadings(input)
	assert.Contains(t, got, "# Title")
	assert.Contains(t, got, "## Subtitle")
	assert.Contains(t, got, "### Deep")
}

func TestNormalizeHeadings_AlreadyH1(t *testing.T) {
	input := "# Title\n\n## Subtitle\n"
	got := normalizeHeadings(input)
	assert.Equal(t, input, got)
}

func TestNormalizeHeadings_NoHeadings(t *testing.T) {
	input := "Just some text without any headings."
	got := normalizeHeadings(input)
	assert.Equal(t, input, got)
}

func TestNormalizeHeadings_MixedLevels(t *testing.T) {
	input := "## Section\n\n#### Subsection\n\n###### Deep\n"
	got := normalizeHeadings(input)
	assert.Contains(t, got, "# Section")
	assert.Contains(t, got, "### Subsection")
	assert.Contains(t, got, "##### Deep")
}

func TestNormalizeHeadings_H2OnlyDocument(t *testing.T) {
	input := "## Only H2 Here\n\nContent.\n"
	got := normalizeHeadings(input)
	assert.Contains(t, got, "# Only H2 Here")
}

func TestStripWordArtifacts_UnderlineSpan(t *testing.T) {
	input := "Some [underlined text]{.underline} here."
	got := stripWordArtifacts(input)
	assert.Equal(t, "Some underlined text here.", got)
}

func TestStripWordArtifacts_EmptySpan(t *testing.T) {
	input := "Before []{.underline} after."
	got := stripWordArtifacts(input)
	assert.Equal(t, "Before  after.", got)
}

func TestStripWordArtifacts_MultipleSpans(t *testing.T) {
	input := "[text1]{.class1} and [text2]{.class2}"
	got := stripWordArtifacts(input)
	assert.Equal(t, "text1 and text2", got)
}

func TestStripWordArtifacts_NoArtifacts(t *testing.T) {
	input := "Normal markdown with [a link](https://example.com)."
	got := stripWordArtifacts(input)
	// Links should not be affected because they use () not {}.
	assert.Equal(t, input, got)
}

func TestConvertImagePaths_AbsoluteToRelative(t *testing.T) {
	input := "![alt](/tmp/images/media/image1.png)"
	got := convertImagePaths(input, "/tmp/images")
	assert.Equal(t, "![alt](images/media/image1.png)", got)
}

func TestConvertImagePaths_AlreadyRelative(t *testing.T) {
	input := "![alt](media/image1.png)"
	got := convertImagePaths(input, "/tmp/images")
	assert.Equal(t, "![alt](media/image1.png)", got)
}

func TestConvertImagePaths_EmptyImagesDir(t *testing.T) {
	input := "![alt](/absolute/path/image.png)"
	got := convertImagePaths(input, "")
	assert.Equal(t, input, got)
}

func TestConvertImagePaths_MultipleImages(t *testing.T) {
	input := "![a](/tmp/images/media/img1.png) and ![b](/tmp/images/media/img2.png)"
	got := convertImagePaths(input, "/tmp/images")
	assert.Contains(t, got, "![a](images/media/img1.png)")
	assert.Contains(t, got, "![b](images/media/img2.png)")
}

func TestClean_FullPipeline(t *testing.T) {
	input := strings.Join([]string{
		"### Title",
		"",
		"",
		"",
		"Some [text]{.underline} content.",
		"",
		"#### Subtitle",
		"",
		"![img](/tmp/media/media/pic.png)",
		"",
		"",
		"",
		"End.",
		"",
	}, "\n")

	got := Clean(input, "/tmp/media")

	// Headings normalized: ### -> #, #### -> ##
	assert.Contains(t, got, "# Title")
	assert.Contains(t, got, "## Subtitle")

	// Word artifacts stripped
	assert.Contains(t, got, "Some text content.")
	assert.NotContains(t, got, "{.underline}")

	// Image paths made relative
	assert.Contains(t, got, "![img](media/media/pic.png)")

	// Excessive blank lines collapsed
	assert.NotContains(t, got, "\n\n\n")
}

func TestClean_EmptyInput(t *testing.T) {
	got := Clean("", "")
	assert.Equal(t, "", got)
}

func TestClean_NoOpWhenAlreadyClean(t *testing.T) {
	input := "# Title\n\nSome text.\n\n## Section\n\nMore text.\n"
	got := Clean(input, "")
	assert.Equal(t, input, got)
}

// TestNormalizeHeadings_PandocSample tests with realistic pandoc output where
// the document starts at H3 (common for Word docs with styled headings).
func TestNormalizeHeadings_PandocSample(t *testing.T) {
	input := `### Introduction

Some introductory text.

#### Background

Background information here.

##### Details

Detailed content.

### Conclusion

Wrap up.
`
	got := normalizeHeadings(input)
	lines := strings.Split(got, "\n")

	// First heading should be H1.
	assert.Equal(t, "# Introduction", lines[0])
	// H4 -> H2
	assert.Equal(t, "## Background", lines[4])
	// H5 -> H3
	assert.Equal(t, "### Details", lines[8])
	// Second H3 -> H1
	assert.Equal(t, "# Conclusion", lines[12])
}

func TestStripWordArtifacts_PreservesMarkdownLinks(t *testing.T) {
	// Standard markdown links use [text](url) - the parentheses distinguish
	// them from Word artifacts which use [text]{.class}.
	input := "See [this link](https://example.com) for details."
	got := stripWordArtifacts(input)
	assert.Equal(t, input, got)
}

func TestStripWordArtifacts_ComplexAttributes(t *testing.T) {
	input := "[styled]{.custom-style style=\"color: red\"}"
	got := stripWordArtifacts(input)
	assert.Equal(t, "styled", got)
}
