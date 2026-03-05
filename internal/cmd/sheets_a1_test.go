package cmd

import "testing"

func TestParseA1Range(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		r, err := parseA1Range("Sheet1!A2:B3")
		if err != nil {
			t.Fatalf("parseA1Range: %v", err)
		}
		if r.SheetName != "Sheet1" || r.StartRow != 2 || r.EndRow != 3 || r.StartCol != 1 || r.EndCol != 2 {
			t.Fatalf("unexpected range: %#v", r)
		}
	})

	t.Run("quoted sheet", func(t *testing.T) {
		r, err := parseA1Range("'My Sheet'!C1:D2")
		if err != nil {
			t.Fatalf("parseA1Range: %v", err)
		}
		if r.SheetName != "My Sheet" || r.StartRow != 1 || r.EndRow != 2 || r.StartCol != 3 || r.EndCol != 4 {
			t.Fatalf("unexpected range: %#v", r)
		}
	})

	t.Run("escaped quote in sheet", func(t *testing.T) {
		r, err := parseA1Range("'Bob''s Sheet'!AA10:AB11")
		if err != nil {
			t.Fatalf("parseA1Range: %v", err)
		}
		if r.SheetName != "Bob's Sheet" || r.StartRow != 10 || r.EndRow != 11 || r.StartCol != 27 || r.EndCol != 28 {
			t.Fatalf("unexpected range: %#v", r)
		}
	})

	t.Run("reordered", func(t *testing.T) {
		r, err := parseA1Range("Sheet1!C3:A1")
		if err != nil {
			t.Fatalf("parseA1Range: %v", err)
		}
		if r.StartRow != 1 || r.EndRow != 3 || r.StartCol != 1 || r.EndCol != 3 {
			t.Fatalf("unexpected range: %#v", r)
		}
	})

	t.Run("invalid cell", func(t *testing.T) {
		if _, err := parseA1Range("Sheet1!A"); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestIndexToColLetters(t *testing.T) {
	tests := []struct {
		col  int
		want string
	}{
		{1, "A"}, {2, "B"}, {26, "Z"}, {27, "AA"}, {52, "AZ"},
	}
	for _, tt := range tests {
		got := indexToColLetters(tt.col)
		if got != tt.want {
			t.Errorf("indexToColLetters(%d)=%q want %q", tt.col, got, tt.want)
		}
	}
	if indexToColLetters(0) != "" {
		t.Errorf("indexToColLetters(0) should be empty")
	}
}

func TestA1RangeString(t *testing.T) {
	got := a1RangeString("Sheet1", 2, 5, 1, 3)
	if got != "Sheet1!A2:C5" {
		t.Errorf("a1RangeString got %q want Sheet1!A2:C5", got)
	}
	gotSingle := a1RangeString("S", 1, 1, 1, 1)
	if gotSingle != "S!A1" {
		t.Errorf("a1RangeString single cell got %q want S!A1", gotSingle)
	}
}
