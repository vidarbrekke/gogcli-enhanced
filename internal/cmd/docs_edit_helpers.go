package cmd

import (
	"strings"

	"google.golang.org/api/docs/v1"
)

// newDocsEditError creates a structured edit error scoped to the Docs service.
func newDocsEditError(op, docID, code, msg string, cause error) error {
	return NewEditError("docs", op, docID, code, msg, cause)
}

// isDocsNotFound checks if an error is a 404 from the Docs API.
func isDocsNotFound(err error) bool {
	return IsNotFound(err)
}

func docsAppendIndex(doc *docs.Document) int64 {
	if doc == nil || doc.Body == nil || len(doc.Body.Content) == 0 {
		return 1
	}
	last := doc.Body.Content[len(doc.Body.Content)-1]
	if last == nil || last.EndIndex <= 1 {
		return 1
	}
	return last.EndIndex - 1
}

type docsTextMatch struct {
	Start int64
	End   int64
}

func docsFindAllTextMatches(doc *docs.Document, needle string) []docsTextMatch {
	needle = strings.TrimSpace(needle)
	if doc == nil || doc.Body == nil || needle == "" {
		return nil
	}
	matches := make([]docsTextMatch, 0)
	for _, se := range doc.Body.Content {
		if se == nil || se.Paragraph == nil {
			continue
		}
		for _, pe := range se.Paragraph.Elements {
			if pe == nil || pe.TextRun == nil {
				continue
			}
			content := pe.TextRun.Content
			if strings.Contains(content, needle) && pe.StartIndex > 0 && pe.EndIndex > pe.StartIndex {
				matches = append(matches, docsTextMatch{Start: pe.StartIndex, End: pe.EndIndex})
			}
		}
	}
	return matches
}

// docsFindTextRanges returns exact (startIndex, endIndex) for each occurrence of needle in doc (UTF-16 indices).
func docsFindTextRanges(doc *docs.Document, needle string) []docsTextMatch {
	needle = strings.TrimSpace(needle)
	if doc == nil || doc.Body == nil || needle == "" {
		return nil
	}
	var matches []docsTextMatch
	for _, se := range doc.Body.Content {
		if se == nil || se.Paragraph == nil {
			continue
		}
		for _, pe := range se.Paragraph.Elements {
			if pe == nil || pe.TextRun == nil {
				continue
			}
			content := pe.TextRun.Content
			runStart := pe.StartIndex
			idx := 0
			for {
				pos := strings.Index(content[idx:], needle)
				if pos < 0 {
					break
				}
				offset := idx + pos
				matchStart := runStart + int64(utf16CodeUnitLen(content[:offset]))
				matchEnd := matchStart + int64(utf16CodeUnitLen(needle))
				matches = append(matches, docsTextMatch{Start: matchStart, End: matchEnd})
				idx = offset + len(needle)
				if idx >= len(content) {
					break
				}
			}
		}
	}
	return matches
}

// utf16CodeUnitLen returns the length of s in UTF-16 code units (Docs API indexing).
func utf16CodeUnitLen(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// docsHeadingEntry is a heading position for positions headings output.
type docsHeadingEntry struct {
	StartIndex int64
	EndIndex   int64
	Style      string
	Text       string
}

// docsHeadingRanges returns heading paragraph positions (HEADING_1..HEADING_6) with optional text snippet.
func docsHeadingRanges(doc *docs.Document) []docsHeadingEntry {
	if doc == nil || doc.Body == nil {
		return nil
	}
	var out []docsHeadingEntry
	for _, se := range doc.Body.Content {
		if se == nil || se.Paragraph == nil {
			continue
		}
		style := ""
		if se.Paragraph.ParagraphStyle != nil && se.Paragraph.ParagraphStyle.NamedStyleType != "" {
			style = se.Paragraph.ParagraphStyle.NamedStyleType
		}
		if !strings.HasPrefix(style, "HEADING_") {
			continue
		}
		start := se.StartIndex
		end := se.EndIndex
		if end <= start {
			continue
		}
		text := ""
		for _, el := range se.Paragraph.Elements {
			if el != nil && el.TextRun != nil && el.TextRun.Content != "" {
				text = strings.TrimSpace(el.TextRun.Content)
				break
			}
		}
		out = append(out, docsHeadingEntry{StartIndex: start, EndIndex: end - 1, Style: style, Text: text})
	}
	return out
}
