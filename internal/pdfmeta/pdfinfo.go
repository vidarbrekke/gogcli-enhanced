package pdfmeta

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

var runPDFInfo = func(ctx context.Context, filePath string) ([]byte, error) {
	// #nosec G204 -- pdfinfo path is intentionally user-provided local file path.
	cmd := exec.CommandContext(ctx, "pdfinfo", filePath)
	return cmd.CombinedOutput()
}

var pdfInfoLine = regexp.MustCompile(`(?im)^\s*pages\s*:\s*(\d+)\s*$`)

//nolint:wsl_v5
func resolvePDFInfo(ctx context.Context, filePath string, run PDFInfoRunner) (int, Attempt) {
	start := time.Now()

	if run == nil {
		run = runPDFInfo
	}

	raw, err := run(ctx, filePath)
	attempt := Attempt{
		Method:     MethodPDFInfo,
		Status:     StatusUnavailable,
		Confidence: confidenceUnreliable,
		DurationMs: elapsedMS(start),
	}

	if err != nil {
		attempt.Reason = err.Error()
		return 0, attempt
	}
	count, ok := parsePDFInfoPages(string(raw))
	if !ok {
		attempt.Status = StatusAmbiguous
		attempt.Reason = "pdfinfo output did not include page count"
		return 0, attempt
	}
	attempt.Status = StatusOK
	attempt.PageCount = count
	attempt.Confidence = confidencePDFInfo
	return count, attempt
}

//nolint:wsl_v5
func parsePDFInfoPages(rawOutput string) (int, bool) {
	match := pdfInfoLine.FindStringSubmatch(rawOutput)
	if len(match) != 2 {
		return 0, false
	}
	pageCount, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	if pageCount < 0 {
		return 0, false
	}

	return pageCount, true
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
