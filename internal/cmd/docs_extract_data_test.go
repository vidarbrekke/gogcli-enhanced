package cmd

import (
	"testing"

	"google.golang.org/api/docs/v1"
)

func TestParseExtractSections(t *testing.T) {
	tests := []struct {
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
	for _, tt := range tests {
		sec := parseExtractSections(tt.in)
		if sec.outline != tt.outline || sec.tables != tt.tables || sec.links != tt.links {
			t.Errorf("parseExtractSections(%q) = outline=%v tables=%v links=%v; want %v %v %v",
				tt.in, sec.outline, sec.tables, sec.links, tt.outline, tt.tables, tt.links)
		}
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
