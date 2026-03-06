package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DiffEntry is a single difference (path + summary).
type DiffEntry struct {
	Path    string
	Summary string
}

// DiffRules configures which paths are drift-only and which arrays to compare set-by-id.
type DiffRules struct {
	// DriftPaths: path suffix or segment that makes a difference drift-only (e.g. "message", "google_reason").
	DriftPaths []string
	// LabelsSetByID: key name for arrays that should be compared as set by "id" (e.g. "labels").
	// Each element is expected to have an "id" field; order is non-contractual.
	LabelsSetByID []string
}

// Diff performs recursive diff of canonical vs baseline. Returns breaking and drift entries.
// Uses json-pointer style paths (e.g. /labels/0/id). Order in arrays is non-contractual when
// the key is in LabelsSetByID (compare by id set).
func Diff(canonical, baseline any, rules DiffRules) (breaking, drift []DiffEntry) {
	var all []struct {
		path    string
		summary string
		isDrift bool
	}
	diffNode("", canonical, baseline, rules, &all)
	for _, c := range all {
		e := DiffEntry{Path: c.path, Summary: c.summary}
		if c.isDrift {
			drift = append(drift, e)
		} else {
			breaking = append(breaking, e)
		}
	}
	// Deterministic output for CI: sort within each bucket.
	sort.Slice(breaking, func(i, j int) bool {
		if breaking[i].Path != breaking[j].Path {
			return breaking[i].Path < breaking[j].Path
		}
		return breaking[i].Summary < breaking[j].Summary
	})
	sort.Slice(drift, func(i, j int) bool {
		if drift[i].Path != drift[j].Path {
			return drift[i].Path < drift[j].Path
		}
		return drift[i].Summary < drift[j].Summary
	})
	return breaking, drift
}

func diffNode(path string, a, b any, rules DiffRules, out *[]struct {
	path    string
	summary string
	isDrift bool
},
) {
	if path != "" && isDriftPath(path, rules.DriftPaths) {
		if !deepEqual(a, b) {
			*out = append(*out, struct {
				path    string
				summary string
				isDrift bool
			}{path, fmt.Sprintf("drift: %v vs %v", a, b), true})
		}
		return
	}

	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			*out = append(*out, struct {
				path    string
				summary string
				isDrift bool
			}{path, "type mismatch: object vs non-object", false})
			return
		}
		allKeys := make(map[string]bool)
		for k := range av {
			allKeys[k] = true
		}
		for k := range bv {
			allKeys[k] = true
		}
		keys := make([]string, 0, len(allKeys))
		for k := range allKeys {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			childPath := path + "/" + k
			ax, aHas := av[k]
			bx, bHas := bv[k]
			if !aHas {
				*out = append(*out, struct {
					path    string
					summary string
					isDrift bool
				}{childPath, "only in baseline", false})
				continue
			}
			if !bHas {
				*out = append(*out, struct {
					path    string
					summary string
					isDrift bool
				}{childPath, "only in canonical", false})
				continue
			}
			diffNode(childPath, ax, bx, rules, out)
		}
	case []any:
		bv, ok := b.([]any)
		if !ok {
			*out = append(*out, struct {
				path    string
				summary string
				isDrift bool
			}{path, "type mismatch: array vs non-array", false})
			return
		}
		keyName := labelsKeyForPath(path, rules.LabelsSetByID)
		if keyName != "" {
			diffLabelsSetByID(path, av, bv, "id", rules, out)
			return
		}
		if len(av) != len(bv) {
			*out = append(*out, struct {
				path    string
				summary string
				isDrift bool
			}{path, fmt.Sprintf("array length %d vs %d", len(av), len(bv)), false})
			return
		}
		for i := range av {
			diffNode(fmt.Sprintf("%s/%d", path, i), av[i], bv[i], rules, out)
		}
	default:
		if !deepEqual(a, b) {
			*out = append(*out, struct {
				path    string
				summary string
				isDrift bool
			}{path, fmt.Sprintf("%v vs %v", a, b), false})
		}
	}
}

func labelsKeyForPath(path string, labelsSetByID []string) string {
	for _, key := range labelsSetByID {
		if path == "" {
			continue
		}
		if path == "/"+key || strings.HasSuffix(path, "/"+key) {
			return key
		}
	}
	return ""
}

func diffLabelsSetByID(path string, a, b []any, idKey string, rules DiffRules, out *[]struct {
	path    string
	summary string
	isDrift bool
},
) {
	am := indexByID(a, idKey)
	bm := indexByID(b, idKey)
	allIDs := make(map[string]bool)
	for id := range am {
		allIDs[id] = true
	}
	for id := range bm {
		allIDs[id] = true
	}
	idsSorted := make([]string, 0, len(allIDs))
	for id := range allIDs {
		idsSorted = append(idsSorted, id)
	}
	sort.Strings(idsSorted)
	for _, id := range idsSorted {
		ax, aHas := am[id]
		bx, bHas := bm[id]
		childPath := path + "/id:" + id
		if !aHas {
			*out = append(*out, struct {
				path    string
				summary string
				isDrift bool
			}{childPath, "only in baseline", false})
			continue
		}
		if !bHas {
			*out = append(*out, struct {
				path    string
				summary string
				isDrift bool
			}{childPath, "only in canonical", false})
			continue
		}
		diffNode(childPath, ax, bx, rules, out)
	}
	// Order difference: if same set of ids but different order, that's drift (already not reported as breaking)
}

func indexByID(arr []any, idKey string) map[string]any {
	out := make(map[string]any)
	for _, v := range arr {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m[idKey].(string)
		if id == "" {
			continue
		}
		out[id] = v
	}
	return out
}

func isDriftPath(path string, driftPaths []string) bool {
	for _, p := range driftPaths {
		if path == p || strings.HasSuffix(path, "/"+p) {
			return true
		}
	}
	return false
}

func deepEqual(a, b any) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
