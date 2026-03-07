package pdfmeta

import "context"

const (
	defaultPDFInfoTailBytes     int64 = 64 * 1024
	defaultRangeXRefBytes       int64 = 128 * 1024
	defaultRangeObjectFetchSize int64 = 8 * 1024
)

const (
	confidencePDFInfo    = 0.99
	confidenceRangeParse = 0.84
	confidenceUnreliable = 0.7
)

type Status string

const (
	StatusOK               Status = "ok"
	StatusAmbiguous        Status = "ambiguous"
	StatusFallbackRequired Status = "fallback_required"
	StatusUnavailable      Status = "unavailable"
	StatusError            Status = "error"
)

type Method string

const (
	MethodPDFInfo     Method = "drive_download_pdfinfo"
	MethodRangeParse  Method = "drive_range_parse"
	MethodThirdParty  Method = "third_party"
	MethodAppsScript  Method = "apps_script_blob_probe"
	MethodUnavailable Method = "unconfigured"
)

type Attempt struct {
	Method     Method  `json:"method"`
	Status     Status  `json:"status"`
	PageCount  int     `json:"page_count,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	Confidence float64 `json:"confidence"`
	DurationMs int64   `json:"duration_ms"`
}

type Result struct {
	PageCount  int       `json:"page_count,omitempty"`
	Status     Status    `json:"status"`
	Source     Method    `json:"source,omitempty"`
	Confidence float64   `json:"confidence"`
	Attempts   []Attempt `json:"attempts"`
}

type RangeClient interface {
	FetchSuffix(ctx context.Context, maxBytes int64) ([]byte, error)
	FetchAt(ctx context.Context, offset int64, length int64) ([]byte, error)
}

type PDFInfoRunner func(ctx context.Context, filePath string) ([]byte, error)

type ResolveOptions struct {
	FilePath         string
	RangeClient      RangeClient
	PDFInfoRunner    PDFInfoRunner
	RangeTailBytes   int64
	RangeXRefBytes   int64
	RangeObjectBytes int64
}

func (o ResolveOptions) normalized() ResolveOptions {
	if o.PDFInfoRunner == nil {
		o.PDFInfoRunner = runPDFInfo
	}

	if o.RangeTailBytes <= 0 {
		o.RangeTailBytes = defaultPDFInfoTailBytes
	}

	if o.RangeXRefBytes <= 0 {
		o.RangeXRefBytes = defaultRangeXRefBytes
	}

	if o.RangeObjectBytes <= 0 {
		o.RangeObjectBytes = defaultRangeObjectFetchSize
	}

	return o
}

func Resolve(ctx context.Context, opts ResolveOptions) Result {
	opts = opts.normalized()
	var attempts []Attempt

	if opts.FilePath != "" {
		count, attempt := resolvePDFInfo(ctx, opts.FilePath, opts.PDFInfoRunner)
		attempts = append(attempts, attempt)

		if attempt.Status == StatusOK {
			return Result{
				PageCount:  count,
				Status:     StatusOK,
				Source:     MethodPDFInfo,
				Confidence: confidencePDFInfo,
				Attempts:   attempts,
			}
		}
	}

	if opts.RangeClient != nil {
		count, attempt := resolveRangeParse(ctx, opts)
		attempts = append(attempts, attempt)

		if attempt.Status == StatusOK {
			return Result{
				PageCount:  count,
				Status:     StatusOK,
				Source:     MethodRangeParse,
				Confidence: attempt.Confidence,
				Attempts:   attempts,
			}
		}
	}

	status := StatusUnavailable
	source := MethodUnavailable
	confidence := 0.0

	for _, a := range attempts {
		if a.Method == MethodUnavailable {
			continue
		}
		source = a.Method

		switch a.Status {
		case StatusAmbiguous:
			status = StatusFallbackRequired
			confidence = a.Confidence
		case StatusError:
			if status == StatusUnavailable {
				status = StatusFallbackRequired
				confidence = 0.0
			}
		}
	}

	return Result{
		Status:     status,
		Source:     source,
		Confidence: confidence,
		Attempts:   attempts,
	}
}
