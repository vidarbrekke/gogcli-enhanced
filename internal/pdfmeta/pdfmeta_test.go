package pdfmeta

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
)

var (
	errPDFInfoMissing      = errors.New("pdfinfo missing")
	errInvalidMaxBytes     = errors.New("invalid maxBytes")
	errInvalidRangeRequest = errors.New("invalid range request")
	errRangeStartTooFar    = errors.New("range start too far")
)

//nolint:wsl_v5
func TestResolvePrefersPDFInfoOverRangeParse(t *testing.T) {
	var pdfInfoCalled int

	options := ResolveOptions{
		FilePath: "document.pdf",
		PDFInfoRunner: func(context.Context, string) ([]byte, error) {
			pdfInfoCalled++
			return []byte("Title: sample\nPages: 11\n"), nil
		},
		RangeClient: &testRangeClient{
			data: mustBuildSyntheticPDF(999),
		},
	}

	got := Resolve(context.Background(), options)
	if got.Status != StatusOK {
		t.Fatalf("status=%s", got.Status)
	}
	if got.Source != MethodPDFInfo {
		t.Fatalf("source=%s", got.Source)
	}
	if got.PageCount != 11 {
		t.Fatalf("pageCount=%d", got.PageCount)
	}
	if pdfInfoCalled != 1 {
		t.Fatalf("pdfinfo run count=%d", pdfInfoCalled)
	}
	if len(got.Attempts) != 1 {
		t.Fatalf("attempts=%d", len(got.Attempts))
	}
}

//nolint:wsl_v5
func TestResolveFallsBackToRangeParse(t *testing.T) {
	options := ResolveOptions{
		FilePath: "document.pdf",
		PDFInfoRunner: func(context.Context, string) ([]byte, error) {
			return nil, errPDFInfoMissing
		},
		RangeClient: &testRangeClient{
			data: mustBuildSyntheticPDF(7),
		},
	}

	got := Resolve(context.Background(), options)
	if got.Status != StatusOK {
		t.Fatalf("status=%s", got.Status)
	}
	if got.Source != MethodRangeParse {
		t.Fatalf("source=%s", got.Source)
	}
	if got.PageCount != 7 {
		t.Fatalf("pageCount=%d", got.PageCount)
	}
	if len(got.Attempts) != 2 {
		t.Fatalf("attempts=%d", len(got.Attempts))
	}
}

//nolint:wsl_v5
func TestResolveWithoutConfiguredStrategies(t *testing.T) {
	got := Resolve(context.Background(), ResolveOptions{})
	if got.Status != StatusUnavailable {
		t.Fatalf("status=%s", got.Status)
	}
	if got.Source != MethodUnavailable {
		t.Fatalf("source=%s", got.Source)
	}
	if len(got.Attempts) != 0 {
		t.Fatalf("attempts=%d", len(got.Attempts))
	}
}

//nolint:wsl_v5
func TestResolveRangeParseExtractsPageCount(t *testing.T) {
	client := &testRangeClient{
		data: mustBuildSyntheticPDF(15),
	}

	gotCount, attempt := resolveRangeParse(context.Background(), ResolveOptions{
		RangeClient:      client,
		RangeTailBytes:   128 * 1024,
		RangeXRefBytes:   128 * 1024,
		RangeObjectBytes: 4096,
	})
	if attempt.Status != StatusOK {
		t.Fatalf("status=%s reason=%s", attempt.Status, attempt.Reason)
	}
	if gotCount != 15 {
		t.Fatalf("count=%d", gotCount)
	}
}

//nolint:wsl_v5
func TestResolveRangeParseAmbiguousWithoutRoot(t *testing.T) {
	pdf := mustBuildSyntheticPDF(15)
	// Remove trailer root and make parse fail cleanly.
	pdf = bytes.ReplaceAll(pdf, []byte("/Root 1 0 R"), []byte("/RooT missing"))

	client := &testRangeClient{data: pdf}
	_, attempt := resolveRangeParse(context.Background(), ResolveOptions{
		RangeClient:      client,
		RangeTailBytes:   128 * 1024,
		RangeXRefBytes:   128 * 1024,
		RangeObjectBytes: 4096,
	})
	if attempt.Status != StatusAmbiguous {
		t.Fatalf("status=%s reason=%s", attempt.Status, attempt.Reason)
	}
}

type testRangeClient struct {
	data      []byte
	suffixCnt int
	atCnt     int
}

//nolint:wsl_v5
func (c *testRangeClient) FetchSuffix(_ context.Context, maxBytes int64) ([]byte, error) {
	c.suffixCnt++
	if maxBytes <= 0 {
		return nil, errInvalidMaxBytes
	}
	if maxBytes >= int64(len(c.data)) {
		return c.data, nil
	}
	return c.data[len(c.data)-int(maxBytes):], nil
}

//nolint:wsl_v5
func (c *testRangeClient) FetchAt(_ context.Context, offset, length int64) ([]byte, error) {
	c.atCnt++
	if offset < 0 || length < 1 {
		return nil, errInvalidRangeRequest
	}
	start := offset
	end := offset + length
	if start > int64(len(c.data)) {
		return nil, errRangeStartTooFar
	}
	if end > int64(len(c.data)) {
		end = int64(len(c.data))
	}
	return c.data[start:end], nil
}

//nolint:wsl_v5
func mustBuildSyntheticPDF(pageCount int) []byte {
	type entry struct {
		id   int
		body string
	}
	entries := []entry{
		{id: 1, body: "/Type /Catalog /Pages 2 0 R"},
		{id: 2, body: fmt.Sprintf("/Type /Pages /Count %d", pageCount)},
		{id: 3, body: "/Type /Page"},
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n%\u00E2\u00E3\u00CF\u00D3\n")

	offsetByID := make(map[int]int64)
	for _, e := range entries {
		offsetByID[e.id] = int64(len(buf.Bytes()))
		fmt.Fprintf(&buf, "%d 0 obj\n<< %s >>\nendobj\n", e.id, e.body)
	}

	xrefPos := int64(len(buf.Bytes()))
	maxID := 4
	fmt.Fprintf(&buf, "xref\n0 %d\n", maxID)
	for i := 0; i < maxID; i++ {
		offset := int64(0)
		usage := "f "
		gen := "65535"
		if i == 1 {
			offset = offsetByID[1]
			usage = "n "
			gen = "00000"
		}
		if i == 2 {
			offset = offsetByID[2]
			usage = "n "
			gen = "00000"
		}
		if i == 3 {
			offset = offsetByID[3]
			usage = "n "
			gen = "00000"
		}
		fmt.Fprintf(&buf, "%010d %s %s \n", offset, gen, usage)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", maxID, xrefPos)
	return buf.Bytes()
}
