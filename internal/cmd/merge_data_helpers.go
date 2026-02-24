package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/drive/v3"
)

type mergeDataErrBuilder func(code, msg string, cause error) error

func loadMergeDataRecords(dataFile string, newErr mergeDataErrBuilder) ([]map[string]any, map[string]any, error) {
	b, readErr := os.ReadFile(dataFile) //nolint:gosec // user-provided path
	if readErr != nil {
		return nil, nil, newErr("input_open_failed", "read data-file failed", readErr)
	}

	var dataRecords []map[string]any
	if jsonErr := json.Unmarshal(b, &dataRecords); jsonErr != nil {
		return nil, nil, newErr("invalid_json", "parse data-file failed", jsonErr)
	}
	if len(dataRecords) == 0 {
		return nil, nil, newErr("invalid_argument", "data-file contains no records", nil)
	}

	sampleRecord := dataRecords[0]
	if len(sampleRecord) == 0 {
		return nil, nil, newErr("invalid_argument", "data records are empty", nil)
	}
	return dataRecords, sampleRecord, nil
}

func buildMergeDataPreview(records []map[string]any, filenameFormat string, includeTimestamp bool, replaceOpName string) []map[string]any {
	previewRecords := records
	if len(previewRecords) > 3 {
		previewRecords = previewRecords[:3]
	}

	operations := make([]map[string]any, 0, len(previewRecords))
	for _, record := range previewRecords {
		filename := FormatMergeFilename(filenameFormat, record, includeTimestamp)
		ops := make([]map[string]any, 0, len(record))
		for key, value := range record {
			ops = append(ops, map[string]any{
				"operation": replaceOpName,
				"find":      fmt.Sprintf("{{%s}}", key),
				"replace":   fmt.Sprintf("%v", value),
			})
		}
		operations = append(operations, map[string]any{
			"filename":   filename,
			"operations": ops,
		})
	}
	return operations
}

func resolveMergeDataOutputFolder(ctx context.Context, driveSvc *drive.Service, templateID, outputFolderID string) string {
	resolved := strings.TrimSpace(outputFolderID)
	if resolved != "" {
		return resolved
	}
	templateMeta, metaErr := driveSvc.Files.Get(templateID).Fields("parents").Context(ctx).Do()
	if metaErr == nil && templateMeta != nil && len(templateMeta.Parents) > 0 {
		return strings.TrimSpace(templateMeta.Parents[0])
	}
	return ""
}
