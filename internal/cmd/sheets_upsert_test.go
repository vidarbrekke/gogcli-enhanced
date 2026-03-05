package cmd

import (
	"testing"
)

func TestParseKeyColumns(t *testing.T) {
	keyCols, err := parseKeyColumns("0,1,2")
	if err != nil {
		t.Fatalf("parseKeyColumns: %v", err)
	}
	if len(keyCols) != 3 || keyCols[0] != 0 || keyCols[1] != 1 || keyCols[2] != 2 {
		t.Fatalf("unexpected keyCols: %v", keyCols)
	}
	if _, err := parseKeyColumns(""); err == nil {
		t.Fatal("expected error for empty key-columns")
	}
	if _, err := parseKeyColumns("0,x"); err == nil {
		t.Fatal("expected error for invalid index")
	}
}

func TestRowKey(t *testing.T) {
	keyCols := []int{0, 1}
	row := []interface{}{"a", "b", "c"}
	if got := rowKey(row, keyCols); got != "a\x00b" {
		t.Errorf("rowKey got %q", got)
	}
	shortRow := []interface{}{"x"}
	if got := rowKey(shortRow, keyCols); got != "" {
		t.Errorf("rowKey short row should be empty, got %q", got)
	}
}

func TestToFloat(t *testing.T) {
	if v, err := toFloat(float64(3.14)); err != nil || v != 3.14 {
		t.Errorf("toFloat(3.14)=%v,%v", v, err)
	}
	if v, err := toFloat("42"); err != nil || v != 42 {
		t.Errorf("toFloat(\"42\")=%v,%v", v, err)
	}
	if _, err := toFloat("x"); err == nil {
		t.Error("toFloat(\"x\") should error")
	}
}
