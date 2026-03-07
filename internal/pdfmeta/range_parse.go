package pdfmeta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	rangeStartXrefPattern = regexp.MustCompile(`(?is)startxref\s+(\d+)`)
	rangeRootPattern      = regexp.MustCompile(`(?i)/Root\s+(\d+)\s+0\s+R`)
	rangeXRefEntryPattern = regexp.MustCompile(`(?m)^\s*(\d+)\s+\d+\s+([fn])\s*$`)
	rangeXRefRangePattern = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s*$`)
	rangePagesRefPattern  = regexp.MustCompile(`(?i)/Pages\s+(\d+)\s+0\s+R`)
	rangeCountPattern     = regexp.MustCompile(`(?i)/Count\s+(\d+)`)
	errTrailerMissingRoot = errors.New("trailer missing /Root")
	errInvalidRootObject  = errors.New("invalid /Root object")
	errNoXRefEntries      = errors.New("no xref entries found")
	errInvalidXRefEntry   = errors.New("invalid xref entry")
)

//nolint:wsl_v5
func resolveRangeParse(ctx context.Context, opts ResolveOptions) (int, Attempt) {
	start := time.Now()

	attempt := Attempt{
		Method:     MethodRangeParse,
		Status:     StatusUnavailable,
		Confidence: confidenceUnreliable,
		DurationMs: elapsedMS(start),
	}

	tail, err := opts.RangeClient.FetchSuffix(ctx, opts.RangeTailBytes)
	if err != nil {
		attempt.Reason = fmt.Sprintf("fetch tail failed: %v", err)
		return 0, attempt
	}

	startXref, ok := parseStartXref(tail)
	if !ok {
		attempt.Status = StatusAmbiguous
		attempt.Reason = "pdf range parse: missing or invalid startxref marker"
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	xrefChunk, err := opts.RangeClient.FetchAt(ctx, startXref, opts.RangeXRefBytes)
	if err != nil {
		attempt.Reason = fmt.Sprintf("fetch xref chunk failed: %v", err)
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	rootObj, xrefEntries, err := parseXRefChunk(xrefChunk)
	if err != nil {
		attempt.Status = StatusAmbiguous
		attempt.Reason = fmt.Sprintf("parse xref failed: %v", err)
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	rootOffset, ok := xrefEntries[rootObj]
	if !ok || rootOffset <= 0 {
		attempt.Status = StatusAmbiguous
		attempt.Reason = fmt.Sprintf("xref missing root object %d", rootObj)
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	catalogBytes, err := opts.RangeClient.FetchAt(ctx, rootOffset, opts.RangeObjectBytes)
	if err != nil {
		attempt.Reason = fmt.Sprintf("fetch catalog object %d failed: %v", rootObj, err)
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	pagesObj, ok := parsePagesRef(catalogBytes)
	if !ok {
		attempt.Status = StatusAmbiguous
		attempt.Reason = "catalog object missing /Pages reference"
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	pagesOffset, ok := xrefEntries[pagesObj]
	if !ok || pagesOffset <= 0 {
		attempt.Status = StatusAmbiguous
		attempt.Reason = fmt.Sprintf("xref missing pages object %d", pagesObj)
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	pagesBytes, err := opts.RangeClient.FetchAt(ctx, pagesOffset, opts.RangeObjectBytes)
	if err != nil {
		attempt.Reason = fmt.Sprintf("fetch pages object %d failed: %v", pagesObj, err)
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	pageCount, ok := parsePagesCount(pagesBytes)
	if !ok {
		attempt.Status = StatusAmbiguous
		attempt.Reason = "pages object missing /Count"
		attempt.DurationMs = elapsedMS(start)
		return 0, attempt
	}

	attempt.PageCount = pageCount
	attempt.Status = StatusOK
	attempt.Confidence = confidenceRangeParse
	attempt.DurationMs = elapsedMS(start)
	return pageCount, attempt
}

//nolint:wsl_v5
func parseStartXref(data []byte) (int64, bool) {
	matches := rangeStartXrefPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return 0, false
	}

	match := matches[len(matches)-1]
	startXref, err := strconv.ParseInt(string(match[1]), 10, 64)
	if err != nil || startXref <= 0 {
		return 0, false
	}
	return startXref, true
}

//nolint:wsl_v5
func parseXRefChunk(data []byte) (int, map[int]int64, error) {
	rootMatch := rangeRootPattern.FindSubmatch(data)
	if len(rootMatch) != 2 {
		return 0, nil, errTrailerMissingRoot
	}

	rootObj, err := strconv.ParseInt(string(rootMatch[1]), 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %s", errInvalidRootObject, err.Error())
	}
	if rootObj < 0 || rootObj > int64(^uint(0)>>1) {
		return 0, nil, fmt.Errorf("%w: %d", errInvalidRootObject, rootObj)
	}

	rootObjID := int(rootObj)
	xrefData := data
	if i := bytes.Index(xrefData, []byte("xref")); i >= 0 {
		xrefData = xrefData[i:]
	}

	lines := bytes.Split(xrefData, []byte("\n"))
	entries := make(map[int]int64, len(lines))
	nextObjectID := -1
	entriesToRead := 0

	for _, line := range lines {
		lineText := strings.TrimSpace(string(line))
		if lineText == "xref" || strings.HasPrefix(lineText, "trailer") {
			nextObjectID = -1
			entriesToRead = 0
			continue
		}

		if entriesToRead > 0 {
			matches := rangeXRefEntryPattern.FindSubmatch(line)
			if len(matches) != 3 {
				continue
			}
			if string(matches[2]) != "n" {
				nextObjectID++
				entriesToRead--
				continue
			}
			entryOffset, err := strconv.ParseInt(string(matches[1]), 10, 64)
			if err != nil {
				return 0, nil, fmt.Errorf("%w: %q", errInvalidXRefEntry, lineText)
			}
			entries[nextObjectID] = entryOffset
			nextObjectID++
			entriesToRead--
			continue
		}

		matches := rangeXRefRangePattern.FindStringSubmatch(lineText)
		if len(matches) != 3 {
			continue
		}
		startObj, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}
		if count <= 0 {
			continue
		}
		nextObjectID = startObj
		entriesToRead = count
	}

	if len(entries) == 0 {
		return 0, nil, errNoXRefEntries
	}
	return rootObjID, entries, nil
}

//nolint:wsl_v5
func parsePagesRef(data []byte) (int, bool) {
	match := rangePagesRefPattern.FindSubmatch(data)
	if len(match) != 2 {
		return 0, false
	}
	pagesObj, err := strconv.Atoi(string(match[1]))
	if err != nil || pagesObj <= 0 {
		return 0, false
	}
	return pagesObj, true
}

//nolint:wsl_v5
func parsePagesCount(data []byte) (int, bool) {
	match := rangeCountPattern.FindSubmatch(data)
	if len(match) != 2 {
		return 0, false
	}
	count, err := strconv.Atoi(string(match[1]))
	if err != nil || count <= 0 {
		return 0, false
	}
	return count, true
}
