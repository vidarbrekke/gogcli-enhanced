// Package google provides MCP tool handlers for Google Docs, Drive, Sheets, Slides.
// sedmat_policy defines risk classification for sedmat expressions used by docs.smartEdit routing.

package google

import (
	"regexp"
	"strings"
)

// RiskLevel is the classification for a set of sed expressions (low = auto-exec ok, high = require plan/confirm).
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// deleteCommandRe matches sed delete command: d/... or d\... etc.
var deleteCommandRe = regexp.MustCompile(`^\s*d\s*[^\w]`)

// clearDocRe matches s/^$// or s/^\s*$// (clear document).
var clearDocRe = regexp.MustCompile(`^\s*s\s*[^\w]\s*\^\s*\$\s*[^\w]\s*[^\w]`)

// tableDeleteRe matches table ref in pattern (|n| or |*|); table delete when replacement empty.
var tableDeleteRe = regexp.MustCompile(`\|[^|]*\|`)

// imageRefRe matches image reference !(n) or !(*) or ![.
var imageRefRe = regexp.MustCompile(`!\(|!\[`)

// mergeSplitRe matches merge/split/unmerge as replacement (lowercase).
var mergeSplitRe = regexp.MustCompile(`\b(merge|unmerge|split)\b`)

// regexMetaRe matches common regex metacharacters in pattern (not exhaustive).
var regexMetaRe = regexp.MustCompile(`[.*+?\[\]\\^$()|{}]`)

// ClassifySedRiskFromExpressions classifies risk from raw sed expression strings.
// Returns risk level and a short reason for the router. Does not parse full expr; uses heuristics.
func ClassifySedRiskFromExpressions(expressions []string) (RiskLevel, string) {
	if len(expressions) == 0 {
		return RiskLow, "no expressions"
	}

	var highReasons []string
	mediumCount := 0
	hasStructural := false

	for _, raw := range expressions {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		// High: delete command
		if deleteCommandRe.MatchString(raw) {
			highReasons = append(highReasons, "delete command")
			continue
		}
		// High: clear document
		if clearDocRe.MatchString(raw) {
			highReasons = append(highReasons, "clear document")
			continue
		}
		// High: merge/split/unmerge (in replacement)
		if mergeSplitRe.MatchString(strings.ToLower(raw)) {
			highReasons = append(highReasons, "merge/split/unmerge")
			continue
		}
		// High: table delete (|n| or |*| with empty repl)
		if tableDeleteRe.MatchString(raw) {
			if isEmptyReplacement(raw) {
				highReasons = append(highReasons, "table delete")
			} else {
				hasStructural = true
			}
			continue
		}
		// High: image ref with empty replacement
		if imageRefRe.MatchString(raw) && isEmptyReplacement(raw) {
			highReasons = append(highReasons, "image delete")
			continue
		}
		// High: row/col op
		if strings.Contains(raw, "rowOp") || strings.Contains(raw, "colOp") {
			highReasons = append(highReasons, "table row/col op")
			continue
		}

		// Medium: regex in pattern (s/.../.../) — but ^ and $ alone are positional (low risk)
		if strings.HasPrefix(raw, "s") && len(raw) > 2 {
			delim := raw[1]
			if delim >= 32 && delim < 127 && delim != '\\' {
				parts := splitByDelimPolicy(raw[2:], delim)
				if len(parts) >= 1 {
					patternPart := parts[0]
					// Positional only: ^ or $ or ^$ → low risk, don't count as regex
					trimPat := strings.TrimSpace(patternPart)
					if trimPat != "^" && trimPat != "$" && trimPat != "^$" && regexMetaRe.MatchString(patternPart) {
						mediumCount++
					}
				}
			}
			// Structural: table cell or image (non-delete already handled)
			if tableDeleteRe.MatchString(raw) || imageRefRe.MatchString(raw) {
				hasStructural = true
			}
			// a/ or i/ (append/insert after/before line)
			if (strings.HasPrefix(raw, "a") || strings.HasPrefix(raw, "i")) && len(raw) >= 2 && !isAlphanumericPolicy(raw[1]) {
				mediumCount++
			}
		}
	}

	if len(highReasons) > 0 {
		return RiskHigh, "high risk: " + strings.Join(highReasons, "; ")
	}
	if len(expressions) > 3 && hasStructural {
		return RiskHigh, "multiple structural expressions require confirmation"
	}
	if mediumCount > 0 || hasStructural {
		return RiskMedium, "medium risk: regex or structural edit"
	}
	return RiskLow, "low risk: plain replace/positional"
}

func isEmptyReplacement(raw string) bool {
	raw = strings.TrimSpace(raw)
	if len(raw) < 4 || raw[0] != 's' {
		return false
	}
	delim := raw[1]
	parts := splitByDelimPolicy(raw[2:], delim)
	if len(parts) < 2 {
		return true
	}
	return strings.TrimSpace(parts[1]) == ""
}

func splitByDelimPolicy(s string, delim byte) []string {
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			cur.WriteByte(s[i+1])
			i++
			continue
		}
		if s[i] == delim {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(s[i])
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func isAlphanumericPolicy(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
