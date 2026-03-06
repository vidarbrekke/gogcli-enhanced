package cmd

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/docs/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// DocsPositionsCmd groups position-helper subcommands.
type DocsPositionsCmd struct {
	End      DocsPositionsEndCmd      `cmd:"" name:"end" help:"Return the append index (1-based position after last content)"`
	Search   DocsPositionsSearchCmd   `cmd:"" name:"search" help:"Search for text and return start/end indices"`
	Headings DocsPositionsHeadingsCmd `cmd:"" name:"headings" help:"Return positions of HEADING_1..HEADING_6 paragraphs"`
}

// DocsPositionsEndCmd returns the document end index for append operations.
type DocsPositionsEndCmd struct {
	DocID string `arg:"" name:"docId" help:"Doc ID"`
}

func (c *DocsPositionsEndCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}
	doc, err := svc.Documents.Get(id).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return errDocNotFound(id)
		}
		return err
	}
	if doc == nil {
		return errDocNotFound(id)
	}
	appendIndex := docsAppendIndex(doc)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
			"docId":       id,
			"appendIndex": appendIndex,
		})
	}
	u := ui.FromContext(ctx)
	u.Out().Printf("appendIndex\t%d", appendIndex)
	return nil
}

// DocsPositionsSearchCmd searches for text and returns matching ranges.
type DocsPositionsSearchCmd struct {
	DocID     string `arg:"" name:"docId" help:"Doc ID"`
	Text      string `name:"text" short:"t" help:"Text to search for" required:""`
	MatchCase bool   `name:"match-case" help:"Case-sensitive match"`
}

func (c *DocsPositionsSearchCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}
	needle := c.Text
	if !c.MatchCase {
		needle = strings.ToLower(needle)
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}
	doc, err := svc.Documents.Get(id).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return errDocNotFound(id)
		}
		return err
	}
	if doc == nil {
		return errDocNotFound(id)
	}
	var matches []docsTextMatch
	if c.MatchCase {
		matches = docsFindTextRanges(doc, c.Text)
	} else {
		matches = docsFindTextRangesCaseInsensitive(doc, needle)
	}
	// Convert to JSON-serializable slice
	ranges := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		ranges = append(ranges, map[string]any{"startIndex": m.Start, "endIndex": m.End})
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
			"docId":  id,
			"text":   c.Text,
			"ranges": ranges,
		})
	}
	u := ui.FromContext(ctx)
	for _, m := range matches {
		u.Out().Printf("%d\t%d", m.Start, m.End)
	}
	return nil
}

// docsFindTextRangesCaseInsensitive finds needle (already lowercased) in doc, returning UTF-16 ranges.
func docsFindTextRangesCaseInsensitive(doc *docs.Document, needleLower string) []docsTextMatch {
	if doc == nil || doc.Body == nil || needleLower == "" {
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
			contentLower := strings.ToLower(content)
			runStart := pe.StartIndex
			idx := 0
			for {
				pos := strings.Index(contentLower[idx:], needleLower)
				if pos < 0 {
					break
				}
				offset := idx + pos
				matchStart := runStart + int64(utf16CodeUnitLen(content[:offset]))
				matchEnd := matchStart + int64(utf16CodeUnitLen(needleLower))
				matches = append(matches, docsTextMatch{Start: matchStart, End: matchEnd})
				idx = offset + len(needleLower)
				if idx >= len(contentLower) {
					break
				}
			}
		}
	}
	return matches
}

// DocsPositionsHeadingsCmd returns heading paragraph positions.
type DocsPositionsHeadingsCmd struct {
	DocID string `arg:"" name:"docId" help:"Doc ID"`
}

func (c *DocsPositionsHeadingsCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}
	doc, err := svc.Documents.Get(id).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return errDocNotFound(id)
		}
		return err
	}
	if doc == nil {
		return errDocNotFound(id)
	}
	headings := docsHeadingRanges(doc)
	items := make([]map[string]any, 0, len(headings))
	for _, h := range headings {
		items = append(items, map[string]any{
			"startIndex": h.StartIndex,
			"endIndex":   h.EndIndex,
			"style":      h.Style,
			"text":       h.Text,
		})
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
			"docId":    id,
			"headings": items,
		})
	}
	u := ui.FromContext(ctx)
	u.Out().Printf("startIndex\tendIndex\tstyle\ttext")
	for _, h := range headings {
		u.Out().Printf("%d\t%d\t%s\t%s", h.StartIndex, h.EndIndex, h.Style, h.Text)
	}
	return nil
}

func errDocNotFound(id string) error {
	return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
}
