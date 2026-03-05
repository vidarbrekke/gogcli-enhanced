package cmd

import (
	"testing"

	"google.golang.org/api/docs/v1"
)

func TestParseExtractSections(t *testing.T) {
	valid := []struct {
		in     string
		outline, tables, links bool
	}{
		{"", true, true, true},
		{"all", true, true, true},
		{"outline", true, false, false},
		{"tables", false, true, false},
		{"links", false, false, true},
		{"outline,tables", true, true, false},
		{"  outline , tables , links  ", true, true, true},
	}
	for _, tt := range valid {
		sec, err := parseExtractSections(tt.in)
		if err != nil {
			t.Errorf("parseExtractSections(%q): unexpected error %v", tt.in, err)
			continue
		}
		if sec.outline != tt.outline || sec.tables != tt.tables || sec.links != tt.links {
			t.Errorf("parseExtractSections(%q) = outline=%v tables=%v links=%v; want %v %v %v",
				tt.in, sec.outline, sec.tables, sec.links, tt.outline, tt.tables, tt.links)
		}
	}
	invalid := []string{"x", "foo", "outline,foo", "x,y", "  "}
	for _, in := range invalid {
		if in == "  " {
			// only spaces: trimmed to "" which is valid (all)
			continue
		}
		_, err := parseExtractSections(in)
		if err == nil {
			t.Errorf("parseExtractSections(%q): expected error", in)
		}
	}
	// "  " alone is valid (trimmed to empty -> all)
	sec, err := parseExtractSections("   ")
	if err != nil {
		t.Errorf("parseExtractSections(%q): %v", "   ", err)
	}
	if !sec.outline || !sec.tables || !sec.links {
		t.Errorf("parseExtractSections(space): want all true, got %+v", sec)
	}
}

func TestExtractOutline_EmptyDoc(t *testing.T) {
	out := extractOutline(&docs.Document{})
	if len(out) != 0 {
		t.Errorf("extractOutline(empty doc) got %d items, want 0", len(out))
	}
}

func TestExtractTables_NoTables(t *testing.T) {
	out := extractTables(&docs.Document{Body: &docs.Body{Content: []*docs.StructuralElement{}}})
	if len(out) != 0 {
		t.Errorf("extractTables(doc with no tables) got %d items, want 0", len(out))
	}
}

func TestCollectLinksFromContent_NestedTable(t *testing.T) {
	// Link inside a table cell: Body -> Table -> Row -> Cell -> Content -> Paragraph -> TextRun with Link
	doc := &docs.Document{
		Body: &docs.Body{
			Content: []*docs.StructuralElement{
				{
					Table: &docs.Table{
						TableRows: []*docs.TableRow{
							{
								TableCells: []*docs.TableCell{
									{
										Content: []*docs.StructuralElement{
											{
												Paragraph: &docs.Paragraph{
													Elements: []*docs.ParagraphElement{
														{
															TextRun: &docs.TextRun{
																Content: "click here",
																TextStyle: &docs.TextStyle{
																	Link: &docs.Link{Url: "https://example.com"},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	links := extractLinks(doc)
	if len(links) != 1 {
		t.Fatalf("extractLinks(nested table): got %d links, want 1", len(links))
	}
	if links[0]["url"] != "https://example.com" || links[0]["text"] != "click here" {
		t.Errorf("extractLinks(nested table): got %v", links[0])
	}
}
