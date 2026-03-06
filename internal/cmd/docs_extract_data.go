package cmd

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/docs/v1"

	"github.com/steipete/gogcli/internal/outfmt"
)

// DocsExtractDataCmd extracts outline (headings), tables, and links from a Google Doc.
type DocsExtractDataCmd struct {
	DocID    string `arg:"" name:"docId" help:"Doc ID"`
	Sections string `name:"sections" help:"Comma-separated: outline,tables,links, or all (default)" default:"all"`
}

func (c *DocsExtractDataCmd) Run(ctx context.Context, flags *RootFlags) error {
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return usage("empty docId")
	}
	sections, err := parseExtractSections(c.Sections)
	if err != nil {
		return err
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}
	doc, err := svc.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", docID)
		}
		return err
	}
	if doc == nil {
		return fmt.Errorf("doc not found")
	}

	out := map[string]any{
		"documentId": docID,
		"title":      doc.Title,
	}
	if sections.outline {
		out["outline"] = extractOutline(doc)
	}
	if sections.tables {
		out["tables"] = extractTables(doc)
	}
	if sections.links {
		out["links"] = extractLinks(doc)
	}

	return outfmt.WriteJSON(ctx, stdoutWriter(ctx), out)
}

type extractSections struct {
	outline, tables, links bool
}

var errExtractSectionsInvalid = fmt.Errorf("valid sections: outline, tables, links, or all")

// parseExtractSections parses --sections and validates. Returns error on unknown tokens or when no section is selected.
func parseExtractSections(s string) (extractSections, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "all" {
		return extractSections{outline: true, tables: true, links: true}, nil
	}
	var sec extractSections
	var unknown []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch part {
		case "outline":
			sec.outline = true
		case "tables":
			sec.tables = true
		case "links":
			sec.links = true
		case "all":
			return extractSections{outline: true, tables: true, links: true}, nil
		default:
			unknown = append(unknown, part)
		}
	}
	if len(unknown) > 0 {
		return extractSections{}, fmt.Errorf("invalid section(s) %q: %w", strings.Join(unknown, ","), errExtractSectionsInvalid)
	}
	if !sec.outline && !sec.tables && !sec.links {
		return extractSections{}, fmt.Errorf("at least one section required: %w", errExtractSectionsInvalid)
	}
	return sec, nil
}

func extractOutline(doc *docs.Document) []map[string]any {
	headings := docsHeadingRanges(doc)
	out := make([]map[string]any, 0, len(headings))
	for _, h := range headings {
		out = append(out, map[string]any{
			"startIndex": h.StartIndex,
			"endIndex":   h.EndIndex,
			"style":      h.Style,
			"text":       h.Text,
		})
	}
	return out
}

func extractTables(doc *docs.Document) []map[string]any {
	tables := collectAllTables(doc)
	out := make([]map[string]any, 0, len(tables))
	for i, t := range tables {
		rows := make([][]string, 0, len(t.TableRows))
		for _, row := range t.TableRows {
			cells := make([]string, 0, len(row.TableCells))
			for _, cell := range row.TableCells {
				text, _, _ := getCellText(cell)
				cells = append(cells, strings.TrimSpace(text))
			}
			rows = append(rows, cells)
		}
		out = append(out, map[string]any{
			"tableIndex": i + 1,
			"rows":       len(rows),
			"cells":      rows,
		})
	}
	return out
}

func extractLinks(doc *docs.Document) []map[string]any {
	if doc == nil || doc.Body == nil {
		return nil
	}
	return collectLinksFromContent(doc.Body.Content)
}

// collectLinksFromContent recursively walks structural elements (body or table cell content) and collects all link entries.
func collectLinksFromContent(content []*docs.StructuralElement) []map[string]any {
	if len(content) == 0 {
		return nil
	}
	var links []map[string]any
	for _, el := range content {
		if el == nil {
			continue
		}
		if el.Paragraph != nil {
			for _, pe := range el.Paragraph.Elements {
				if pe.TextRun != nil && pe.TextRun.TextStyle != nil && pe.TextRun.TextStyle.Link != nil {
					link := pe.TextRun.TextStyle.Link
					entry := map[string]any{"text": strings.TrimSpace(pe.TextRun.Content)}
					if link.Url != "" {
						entry["url"] = link.Url
					}
					if link.BookmarkId != "" {
						entry["bookmarkId"] = link.BookmarkId
					}
					links = append(links, entry)
				}
			}
		}
		if el.Table != nil {
			for _, row := range el.Table.TableRows {
				for _, cell := range row.TableCells {
					if cell != nil {
						links = append(links, collectLinksFromContent(cell.Content)...)
					}
				}
			}
		}
	}
	return links
}
