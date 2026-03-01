package google

import (
	"strings"
	"testing"
)

func TestClassifySedRiskFromExpressions(t *testing.T) {
	tests := []struct {
		name        string
		expressions []string
		wantLevel   RiskLevel
		wantReason  string
	}{
		{"no expressions", nil, RiskLow, "no expressions"},
		{"empty slice", []string{}, RiskLow, "no expressions"},
		{"plain replace", []string{"s/foo/bar/"}, RiskLow, "low risk"},
		{"plain replace global", []string{"s/foo/bar/g"}, RiskLow, "low risk"},
		{"prepend", []string{"s/^/prefix/"}, RiskLow, "low risk"},
		{"append", []string{"s/$/suffix/"}, RiskLow, "low risk"},
		{"delete command", []string{"d/delete-me/"}, RiskHigh, "high risk"},
		{"clear document", []string{"s/^$//"}, RiskHigh, "high risk"},
		{"table delete", []string{"s/|1|//"}, RiskHigh, "table delete"},
		{"image delete", []string{"s/!(1)//"}, RiskHigh, "image delete"},
		{"merge op", []string{"s/|1|[1,1]/merge/"}, RiskHigh, "merge/split/unmerge"},
		{"regex pattern", []string{"s/\\d+/N/"}, RiskMedium, "medium risk"},
		{"multiple plain", []string{"s/a/b/", "s/c/d/"}, RiskLow, "low risk"},
		{"multiple with delete", []string{"s/foo/bar/", "d/x/"}, RiskHigh, "delete command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLevel, gotReason := ClassifySedRiskFromExpressions(tt.expressions)
			if gotLevel != tt.wantLevel {
				t.Errorf("ClassifySedRiskFromExpressions() level = %v, want %v", gotLevel, tt.wantLevel)
			}
			if tt.wantReason != "" && !strings.Contains(gotReason, tt.wantReason) {
				t.Errorf("ClassifySedRiskFromExpressions() reason = %q, want substring %q", gotReason, tt.wantReason)
			}
		})
	}
}

