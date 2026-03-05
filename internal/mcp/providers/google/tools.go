//nolint:wsl_v5 // command argument composition is intentionally linear
package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/steipete/gogcli/internal/mcp/server"
)

type Executor func(args []string) (stdout string, stderr string, err error)

var (
	errMissingDocID              = errors.New("missing docId")
	errMissingSpreadsheetID      = errors.New("missing spreadsheetId")
	errMissingPresentationID     = errors.New("missing presentationId")
	errMissingRequest            = errors.New("missing request")
	errMissingPath               = errors.New("missing path")
	errMissingFileID             = errors.New("missing fileId")
	errMissingText               = errors.New("missing text")
	errMissingFind               = errors.New("missing find text")
	errMissingRange              = errors.New("missing range")
	errMissingIndex              = errors.New("invalid index")
	errMissingLocalPath          = errors.New("missing localPath")
	errMissingQuery              = errors.New("missing query")
	errMissingFileOrPermissionID = errors.New("missing fileId or permissionId")
	errToolCommandFailed         = errors.New("tool command failed")
	errToolStderr                = errors.New("tool stderr")
	errExecutorNotConfigured     = errors.New("executor not configured")
)

// mcpDrivePageSize is the default page size for drive list/search when called from MCP.
// Chosen so responses fit gateway/exec result length limits; agent uses nextPageToken for more.
const mcpDrivePageSize = 25

// mcpDriveMaxCap caps maxResults/pageSize so response still fits gateway.
const mcpDriveMaxCap = 100

// driveListSearchNormalizeInput copies Drive API-style args into our names so agents can use either.
// pageToken → page; maxResults or pageSize → max (capped by mcpDriveMaxCap).
func driveListSearchNormalizeInput(input map[string]any) {
	page := strings.TrimSpace(asString(input["page"]))
	if page == "" || strings.EqualFold(page, "null") {
		if pt := strings.TrimSpace(asString(input["pageToken"])); pt != "" && !strings.EqualFold(pt, "null") {
			input["page"] = pt
		}
	}
	if _, hasMax := input["max"]; hasMax {
		return
	}
	capVal := int64(mcpDriveMaxCap)
	if n, ok := asInt(input["maxResults"]); ok && n > 0 {
		if n > capVal {
			n = capVal
		}
		input["max"] = n
		return
	}
	if n, ok := asInt(input["pageSize"]); ok && n > 0 {
		if n > capVal {
			n = capVal
		}
		input["max"] = n
	}
}

func Register(s *server.Server, executor Executor) {
	p := &provider{exec: executor}
	// Tool specs are split by domain (docs, sheets, slides, drive, gmail, calendar, contacts); see *_tools.go.
	toolSpecs := append(append(append(append(append(append(append([]server.ToolSpec{},
		docsSpecs(p)...),
		sheetsSpecs(p)...),
		slidesSpecs(p)...),
		driveSpecs(p)...),
		gmailSpecs(p)...),
		calendarSpecs(p)...),
		contactsSpecs(p)...)
	for _, spec := range toolSpecs {
		s.RegisterToolSpec(spec)
	}
}

type provider struct {
	exec Executor
}

func (p *provider) docsPlanBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "docs", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	reqForFile := injectRequireRevision(requests, asString(input["requireRevisionId"]))
	path, err := writeTempJSON(reqForFile)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := make([]string, 0, 12)
	args = append(args, "--json")
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "batch", docID, "--requests-file", path, "--validate-only")
	return p.runCLI(cleanArgs(args), "docs", "planBatch")
}

func (p *provider) docsExecuteBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "docs", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	reqForFile := injectRequireRevision(requests, asString(input["requireRevisionId"]))
	path, err := writeTempJSON(reqForFile)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := make([]string, 0, 12)
	args = append(args, "--json")
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "batch", docID, "--requests-file", path)
	return p.runCLI(cleanArgs(args), "docs", "executeBatch")
}

func (p *provider) docsSed(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "sed", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	var exprs []string
	if e := strings.TrimSpace(asString(input["expression"])); e != "" {
		exprs = append(exprs, e)
	}
	if arr, ok := input["expressions"].([]any); ok {
		for _, v := range arr {
			if s := strings.TrimSpace(asString(v)); s != "" {
				exprs = append(exprs, s)
			}
		}
	}
	if len(exprs) == 0 {
		return map[string]any{"service": "docs", "operation": "sed", "error_code": server.ErrorCodeInvalidArgument, "message": "missing expression or expressions"}, errors.New("missing expression or expressions")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "sed", docID)
	if asBool(input["dryRun"]) {
		args = append(args, "-n")
	}
	if len(exprs) == 1 {
		args = append(args, exprs[0])
	} else {
		args = append(args, exprs[0])
		for _, e := range exprs[1:] {
			args = append(args, "-e", e)
		}
	}
	result, err := p.runCLI(cleanArgs(args), "docs", "sed")
	if err != nil {
		return result, err
	}
	if result != nil {
		result["engine"] = "sedmat"
	}
	return result, nil
}

func (p *provider) docsSmartEdit(ctx context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	intentType := strings.TrimSpace(strings.ToLower(asString(input["intentType"])))
	if intentType == "" {
		intentType = "sed"
	}
	validateOnly := asBool(input["validateOnly"])

	var expressions []string
	if arr, ok := input["expressions"].([]any); ok {
		for _, v := range arr {
			if s := strings.TrimSpace(asString(v)); s != "" {
				expressions = append(expressions, s)
			}
		}
	}

	// Route by intent
	switch intentType {
	case "batch":
		if req, ok := input["request"].(map[string]any); ok {
			if validateOnly {
				return p.docsPlanBatch(ctx, map[string]any{"docId": docID, "request": req, "opId": input["opId"], "timeoutMs": input["timeoutMs"], "retries": input["retries"], "retryBackoffMs": input["retryBackoffMs"]})
			}
			return p.docsExecuteBatch(ctx, map[string]any{"docId": docID, "request": req, "opId": input["opId"], "timeoutMs": input["timeoutMs"], "retries": input["retries"], "retryBackoffMs": input["retryBackoffMs"]})
		}
		return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "intentType batch requires request"}, errMissingRequest
	case "sed":
		if len(expressions) == 0 {
			return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "intentType sed requires expressions"}, errors.New("missing expressions")
		}
		riskLevel, decisionReason := ClassifySedRiskFromExpressions(expressions)
		requiresConfirmation := riskLevel == RiskHigh
		// If validateOnly or high risk, return assessment without executing write
		if validateOnly || (riskLevel == RiskHigh) {
			out := map[string]any{
				"service":              "docs",
				"operation":            "smartEdit",
				"engineSelected":       "sed",
				"decisionReason":       decisionReason,
				"riskLevel":            string(riskLevel),
				"requiresConfirmation": requiresConfirmation,
				"docId":                docID,
			}
			if input["opId"] != nil {
				out["opId"] = asString(input["opId"])
			}
			return out, nil
		}
		// Medium/low: execute sed and wrap result with routing envelope
		sedInput := map[string]any{
			"docId": docID, "opId": input["opId"],
			"timeoutMs": input["timeoutMs"], "retries": input["retries"], "retryBackoffMs": input["retryBackoffMs"],
			"account": input["account"], "dryRun": false,
		}
		if len(expressions) == 1 {
			sedInput["expression"] = expressions[0]
		} else {
			sedInput["expressions"] = toAnySlice(expressions)
		}
		result, err := p.docsSed(ctx, sedInput)
		if err != nil {
			return result, err
		}
		if result != nil {
			result["engineSelected"] = "sed"
			result["decisionReason"] = decisionReason
			result["riskLevel"] = string(riskLevel)
			result["requiresConfirmation"] = false
		}
		return result, nil
	default:
		return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "intentType must be batch or sed"}, errors.New("invalid intentType")
	}
}

func toAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func (p *provider) docsCreate(_ context.Context, input map[string]any) (map[string]any, error) {
	title := strings.TrimSpace(asString(input["title"]))
	if title == "" {
		return map[string]any{"service": "docs", "operation": "create", "error_code": server.ErrorCodeInvalidArgument, "message": "missing title"}, errors.New("missing title")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "create", title)
	if parentID := strings.TrimSpace(asString(input["parentId"])); parentID != "" {
		args = append(args, "--parent", parentID)
	}
	return p.runCLI(cleanArgs(args), "docs", "create")
}

func (p *provider) docsCreateWithBody(ctx context.Context, input map[string]any) (map[string]any, error) {
	createResult, err := p.docsCreate(ctx, input)
	if err != nil {
		return createResult, err
	}
	// Parse docId from create output: gog docs create --json returns {"file": {"id": "...", ...}}
	var docID string
	if fileObj, ok := createResult["file"].(map[string]any); ok {
		docID = asString(fileObj["id"])
	}
	if docID != "" {
		createResult["documentId"] = docID
	}
	if docID == "" {
		return createResult, nil
	}
	reqObj, ok := input["request"].(map[string]any)
	if !ok || reqObj == nil {
		return createResult, nil
	}
	// Apply batchUpdate in same tool call to save a round-trip
	batchInput := map[string]any{"docId": docID, "request": reqObj}
	for _, k := range []string{"account", "opId", "timeoutMs", "retries", "retryBackoffMs"} {
		if v, ok := input[k]; ok {
			batchInput[k] = v
		}
	}
	batchResult, batchErr := p.docsExecuteBatch(ctx, batchInput)
	if batchErr != nil {
		createResult["service"] = "docs"
		createResult["operation"] = "createWithBody"
		createResult["documentId"] = docID
		createResult["batchError"] = batchResult
		return createResult, batchErr
	}
	createResult["documentId"] = docID
	createResult["batch"] = batchResult
	return createResult, nil
}

func (p *provider) docsInsertText(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	text := strings.TrimSpace(asString(input["text"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "insertText", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if text == "" {
		return map[string]any{"service": "docs", "operation": "insertText", "error_code": "invalid_argument", "message": "missing text"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "insert", docID, text)
	if idx, ok := asInt(input["index"]); ok {
		if idx < 1 {
			return map[string]any{"service": "docs", "operation": "insertText", "error_code": "invalid_argument", "message": "index must be >= 1"}, errMissingIndex
		}
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "insertText")
}

func (p *provider) docsDeleteRange(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	start, okStart := asInt(input["startIndex"])
	end, okEnd := asInt(input["endIndex"])
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "deleteRange", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if !okStart || !okEnd {
		return map[string]any{"service": "docs", "operation": "deleteRange", "error_code": "invalid_argument", "message": "missing startIndex/endIndex"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "delete", docID, strconv.FormatInt(start, 10), strconv.FormatInt(end, 10))
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "deleteRange")
}

func (p *provider) docsReplaceAllText(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	find := strings.TrimSpace(asString(input["find"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "replaceAllText", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if find == "" {
		return map[string]any{"service": "docs", "operation": "replaceAllText", "error_code": "invalid_argument", "message": "missing find"}, errMissingFind
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "replace", docID, find, asString(input["replace"]))
	if asBool(input["matchCase"]) {
		args = append(args, "--match-case")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "replaceAllText")
}

func (p *provider) docsAppendText(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	text := strings.TrimSpace(asString(input["text"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "appendText", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if text == "" {
		return map[string]any{"service": "docs", "operation": "appendText", "error_code": "invalid_argument", "message": "missing text"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "append", docID, text)
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "appendText")
}

func (p *provider) docsInsertTable(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "insertTable", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "insert-table", docID)
	if rows, ok := asInt(input["rows"]); ok {
		args = append(args, "--rows", strconv.FormatInt(rows, 10))
	}
	if cols, ok := asInt(input["cols"]); ok {
		args = append(args, "--cols", strconv.FormatInt(cols, 10))
	}
	if idx, ok := asInt(input["index"]); ok {
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "insertTable")
}

func (p *provider) docsInsertImage(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	uri := strings.TrimSpace(asString(input["uri"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "insertImage", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if uri == "" {
		return map[string]any{"service": "docs", "operation": "insertImage", "error_code": "invalid_argument", "message": "missing uri"}, errMissingPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "insert-image", docID, uri)
	if idx, ok := asInt(input["index"]); ok {
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if width, ok := asFloat(input["widthPt"]); ok {
		args = append(args, "--width-pt", trimFloat(width))
	}
	if height, ok := asFloat(input["heightPt"]); ok {
		args = append(args, "--height-pt", trimFloat(height))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "insertImage")
}

func (p *provider) docsLocatorEdit(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "locatorEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "locator", docID)
	if v := strings.TrimSpace(asString(input["after"])); v != "" {
		args = append(args, "--after", v)
	}
	if v := asString(input["insertText"]); v != "" {
		args = append(args, "--insert", v)
	}
	if v := strings.TrimSpace(asString(input["betweenStart"])); v != "" {
		args = append(args, "--between-start", v)
	}
	if v := strings.TrimSpace(asString(input["betweenEnd"])); v != "" {
		args = append(args, "--between-end", v)
	}
	if v := asString(input["replaceText"]); v != "" {
		args = append(args, "--replace", v)
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "locatorEdit")
}

func (p *provider) docsMergeData(_ context.Context, input map[string]any) (map[string]any, error) {
	templateID := strings.TrimSpace(asString(input["templateId"]))
	if templateID == "" {
		return map[string]any{"service": "docs", "operation": "mergeData", "error_code": server.ErrorCodeInvalidArgument, "message": "missing templateId"}, errMissingDocID
	}
	data, ok := input["data"].([]any)
	if !ok || len(data) == 0 {
		return map[string]any{"service": "docs", "operation": "mergeData", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty data array"}, errMissingRequest
	}
	path, err := writeTempJSON(data)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "edit", "merge-data", templateID, "--data-file", path)
	if v := strings.TrimSpace(asString(input["filenameFormat"])); v != "" {
		args = append(args, "--filename-format", v)
	}
	if v := strings.TrimSpace(asString(input["outputFolderId"])); v != "" {
		args = append(args, "--output-folder-id", v)
	}
	if asBool(input["includeTimestamp"]) {
		args = append(args, "--include-timestamp")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	if asBool(input["dryRun"]) {
		args = append(args, "--dry-run")
	}
	return p.runCLI(cleanArgs(args), "docs", "mergeData")
}

func (p *provider) docsGet(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "get", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := make([]string, 0, 8)
	args = append(args, "--json")
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "info", docID)
	return p.runCLI(cleanArgs(args), "docs", "get")
}

func (p *provider) docsCat(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "cat", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "cat", docID)
	if maxBytes, ok := asInt(input["maxBytes"]); ok && maxBytes > 0 {
		args = append(args, "--max-bytes", strconv.FormatInt(maxBytes, 10))
	}
	if tab := strings.TrimSpace(asString(input["tab"])); tab != "" {
		args = append(args, "--tab", tab)
	}
	if asBool(input["allTabs"]) {
		args = append(args, "--all-tabs")
	}
	return p.runCLI(cleanArgs(args), "docs", "cat")
}

func (p *provider) docsListTabs(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "listTabs", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "list-tabs", docID)
	return p.runCLI(cleanArgs(args), "docs", "listTabs")
}

func (p *provider) docsPositionsEnd(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "positionsEnd", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "positions", "end", docID)
	return p.runCLI(cleanArgs(args), "docs", "positionsEnd")
}

func (p *provider) docsPositionsSearch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	text := strings.TrimSpace(asString(input["text"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "positionsSearch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	if text == "" {
		return map[string]any{"service": "docs", "operation": "positionsSearch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing text"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "positions", "search", docID, "--text", text)
	if asBool(input["matchCase"]) {
		args = append(args, "--match-case")
	}
	return p.runCLI(cleanArgs(args), "docs", "positionsSearch")
}

func (p *provider) docsPositionsHeadings(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "positionsHeadings", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "positions", "headings", docID)
	return p.runCLI(cleanArgs(args), "docs", "positionsHeadings")
}

func (p *provider) docsExport(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "export", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	format := strings.TrimSpace(asString(input["format"]))
	if format == "" {
		format = "pdf"
	}
	outPath := strings.TrimSpace(asString(input["out"]))
	if outPath == "" {
		ext := ".pdf"
		switch strings.ToLower(format) {
		case "docx":
			ext = ".docx"
		case "txt":
			ext = ".txt"
		}
		f, tmpErr := os.CreateTemp("", "gog-doc-*"+ext)
		if tmpErr != nil {
			return nil, fmt.Errorf("create temp path: %w", tmpErr)
		}
		outPath = f.Name()
		_ = f.Close()
		_ = os.Remove(outPath)
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "export", docID, "--format", format, "--out", outPath)
	return p.runCLI(cleanArgs(args), "docs", "export")
}

func (p *provider) sheetsPlanBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "sheets", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "batch", spreadsheetID, "--requests-file", path, "--validate-only")
	return p.runCLI(cleanArgs(args), "sheets", "planBatch")
}

func (p *provider) sheetsExecuteBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "sheets", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "batch", spreadsheetID, "--requests-file", path)
	return p.runCLI(cleanArgs(args), "sheets", "executeBatch")
}

func (p *provider) sheetsValuesUpdate(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "valuesUpdate", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "valuesUpdate", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	values, ok := input["values"]
	if !ok {
		return map[string]any{"service": "sheets", "operation": "valuesUpdate", "error_code": server.ErrorCodeInvalidArgument, "message": "missing values"}, errMissingRequest
	}
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal values: %w", err)
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "values", spreadsheetID, rangeSpec, "--values-json", string(valuesJSON))
	if v := strings.TrimSpace(asString(input["valueInput"])); v != "" {
		args = append(args, "--input", v)
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "sheets", "valuesUpdate")
}

func (p *provider) sheetsValuesAppend(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "valuesAppend", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "valuesAppend", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	values, ok := input["values"]
	if !ok {
		return map[string]any{"service": "sheets", "operation": "valuesAppend", "error_code": server.ErrorCodeInvalidArgument, "message": "missing values"}, errMissingRequest
	}
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal values: %w", err)
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "append", spreadsheetID, rangeSpec, "--values-json", string(valuesJSON))
	if v := strings.TrimSpace(asString(input["valueInput"])); v != "" {
		args = append(args, "--input", v)
	}
	if v := strings.TrimSpace(asString(input["insert"])); v != "" {
		args = append(args, "--insert", v)
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "sheets", "valuesAppend")
}

func (p *provider) sheetsLinks(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "links", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "links", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "links", spreadsheetID, rangeSpec)
	return p.runCLI(cleanArgs(args), "sheets", "links")
}

func (p *provider) sheetsValuesGet(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "valuesGet", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "valuesGet", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "get", spreadsheetID, rangeSpec)
	if v := strings.TrimSpace(asString(input["majorDimension"])); v != "" {
		args = append(args, "--dimension", v)
	}
	if v := strings.TrimSpace(asString(input["valueRenderOption"])); v != "" {
		args = append(args, "--render", v)
	}
	return p.runCLI(cleanArgs(args), "sheets", "valuesGet")
}

func (p *provider) sheetsSortRange(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "sortRange", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "sortRange", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	byColumn := int64(0)
	if n, ok := asInt(input["sortByColumn"]); ok {
		byColumn = n
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "sort", spreadsheetID, rangeSpec)
	if byColumn != 0 {
		args = append(args, "--by-column", strconv.FormatInt(byColumn, 10))
	}
	if asBool(input["desc"]) {
		args = append(args, "--desc")
	}
	return p.runCLI(cleanArgs(args), "sheets", "sortRange")
}

func (p *provider) sheetsDedupeRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "dedupeRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "dedupeRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "dedupe", spreadsheetID, rangeSpec)
	if keep := strings.TrimSpace(asString(input["keep"])); keep != "" {
		args = append(args, "--keep", keep)
	}
	if keyCols, ok := input["keyColumns"]; ok {
		switch v := keyCols.(type) {
		case []any:
			var parts []string
			for _, x := range v {
				if n, ok := asInt(x); ok {
					parts = append(parts, strconv.FormatInt(n, 10))
				}
			}
			if len(parts) > 0 {
				args = append(args, "--key-columns", strings.Join(parts, ","))
			}
		case []int:
			parts := make([]string, 0, len(v))
			for _, n := range v {
				parts = append(parts, strconv.Itoa(n))
			}
			if len(parts) > 0 {
				args = append(args, "--key-columns", strings.Join(parts, ","))
			}
		}
	}
	return p.runCLI(cleanArgs(args), "sheets", "dedupeRows")
}

func (p *provider) sheetsFilterCopyRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	targetSheet := strings.TrimSpace(asString(input["targetSheet"]))
	value := asString(input["value"])
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "filterCopyRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "filterCopyRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	if targetSheet == "" {
		return map[string]any{"service": "sheets", "operation": "filterCopyRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing targetSheet"}, errors.New("missing targetSheet")
	}
	if errMap, err := validateSheetOp(input, "op", "sheets", "filterCopyRows"); err != nil {
		return errMap, err
	}
	col, errMap, err := validateSheetColumn(input, "column", "sheets", "filterCopyRows")
	if err != nil {
		return errMap, err
	}
	op := strings.TrimSpace(strings.ToLower(asString(input["op"])))
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "filter-copy", spreadsheetID, rangeSpec, targetSheet, "--column", strconv.FormatInt(col, 10), "--op", op, "--value", value)
	if dest := strings.TrimSpace(asString(input["destinationCell"])); dest != "" {
		args = append(args, "--destination-cell", dest)
	}
	return p.runCLI(cleanArgs(args), "sheets", "filterCopyRows")
}

func (p *provider) sheetsUpsertRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	rows, ok := input["rows"].([]any)
	if !ok || len(rows) == 0 {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty rows"}, errors.New("missing or empty rows")
	}
	keyCols, ok := input["keyColumns"].([]any)
	if !ok || len(keyCols) == 0 {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty keyColumns"}, errors.New("missing or empty keyColumns")
	}
	keyIndices, errMap, err := validateRequiredIntSlice(input, "keyColumns", "sheets", "upsertRows")
	if err != nil {
		return errMap, err
	}
	keyParts := make([]string, 0, len(keyIndices))
	for _, n := range keyIndices {
		keyParts = append(keyParts, strconv.FormatInt(n, 10))
	}
	rowsJSON, err := json.Marshal(input["rows"])
	if err != nil {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "invalid rows"}, fmt.Errorf("marshal rows: %w", err)
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "upsert", spreadsheetID, rangeSpec, "--key-columns", strings.Join(keyParts, ","), "--rows-json", string(rowsJSON))
	return p.runCLI(cleanArgs(args), "sheets", "upsertRows")
}

func (p *provider) sheetsMoveRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	targetSheet := strings.TrimSpace(asString(input["targetSheet"]))
	op := strings.TrimSpace(asString(input["op"]))
	value := asString(input["value"])
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "moveRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "moveRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	if targetSheet == "" {
		return map[string]any{"service": "sheets", "operation": "moveRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing targetSheet"}, errors.New("missing targetSheet")
	}
	if op == "" {
		op = "eq"
	} else {
		if errMap, err := validateSheetOp(input, "op", "sheets", "moveRows"); err != nil {
			return errMap, err
		}
		op = strings.TrimSpace(strings.ToLower(asString(input["op"])))
	}
	col, errMap, err := validateSheetColumn(input, "column", "sheets", "moveRows")
	if err != nil {
		return errMap, err
	}
	mode := strings.TrimSpace(asString(input["mode"]))
	if mode == "" {
		mode = "copy"
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "move-rows", spreadsheetID, rangeSpec, targetSheet, "--column", strconv.FormatInt(col, 10), "--op", op, "--value", value, "--mode", mode)
	if dest := strings.TrimSpace(asString(input["destinationCell"])); dest != "" {
		args = append(args, "--destination-cell", dest)
	}
	return p.runCLI(cleanArgs(args), "sheets", "moveRows")
}

func (p *provider) sheetsApplyFormula(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	formula := strings.TrimSpace(asString(input["formula"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "applyFormula", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "applyFormula", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	if formula == "" {
		return map[string]any{"service": "sheets", "operation": "applyFormula", "error_code": server.ErrorCodeInvalidArgument, "message": "missing formula"}, errors.New("missing formula")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "apply-formula", spreadsheetID, rangeSpec, "--formula", formula)
	return p.runCLI(cleanArgs(args), "sheets", "applyFormula")
}

func (p *provider) sheetsSummarize(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "summarize", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "summarize", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	groupBy, errMap, err := validateRequiredIntSlice(input, "groupBy", "sheets", "summarize")
	if err != nil {
		return errMap, err
	}
	agg := strings.TrimSpace(asString(input["aggregate"]))
	if agg == "" {
		agg = "count"
	}
	groupParts := make([]string, 0, len(groupBy))
	for _, n := range groupBy {
		groupParts = append(groupParts, strconv.FormatInt(n, 10))
	}
	metricCol, _ := asInt(input["metricColumn"])
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "summarize", spreadsheetID, rangeSpec, "--group-by", strings.Join(groupParts, ","), "--metric-column", strconv.FormatInt(metricCol, 10), "--aggregate", agg)
	if target := strings.TrimSpace(asString(input["targetSheet"])); target != "" {
		args = append(args, "--target-sheet", target)
	}
	return p.runCLI(cleanArgs(args), "sheets", "summarize")
}

func (p *provider) sheetsClear(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "clear", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "clear", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "clear", spreadsheetID, rangeSpec)
	if asBool(input["dryRun"]) {
		args = append(args, "--dry-run")
	}
	return p.runCLI(cleanArgs(args), "sheets", "clear")
}

func (p *provider) sheetsMetadata(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "metadata", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "metadata", spreadsheetID)
	return p.runCLI(cleanArgs(args), "sheets", "metadata")
}

func (p *provider) slidesPlanBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "slides", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "batch", presentationID, "--requests-file", path, "--validate-only")
	return p.runCLI(cleanArgs(args), "slides", "planBatch")
}

func (p *provider) slidesExecuteBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "slides", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "batch", presentationID, "--requests-file", path)
	return p.runCLI(cleanArgs(args), "slides", "executeBatch")
}

func (p *provider) slidesReplaceText(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	find := strings.TrimSpace(asString(input["find"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "replaceText", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	if find == "" {
		return map[string]any{"service": "slides", "operation": "replaceText", "error_code": server.ErrorCodeInvalidArgument, "message": "missing find"}, errMissingFind
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "replace-text", presentationID, "--find", find)
	if v := asString(input["replace"]); v != "" {
		args = append(args, "--replace", v)
	}
	if asBool(input["matchCase"]) {
		args = append(args, "--match-case")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "slides", "replaceText")
}

func (p *provider) slidesCreateSlide(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "createSlide", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "create-slide", presentationID)
	if v := strings.TrimSpace(asString(input["layout"])); v != "" {
		args = append(args, "--layout", v)
	}
	if idx, ok := asInt(input["index"]); ok {
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "slides", "createSlide")
}

func (p *provider) driveEnsureFolder(_ context.Context, input map[string]any) (map[string]any, error) {
	path := strings.TrimSpace(asString(input["path"]))
	if path == "" {
		return map[string]any{"service": "drive", "operation": "ensureFolder", "error_code": "invalid_argument", "message": "missing path"}, errMissingPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "ensure-folder", path)
	if parent := strings.TrimSpace(asString(input["parentId"])); parent != "" {
		args = append(args, "--parent", parent)
	}
	return p.runCLI(cleanArgs(args), "drive", "ensureFolder")
}

func (p *provider) driveUntrash(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "untrash", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "untrash", fileID)
	return p.runCLI(cleanArgs(args), "drive", "untrash")
}

func (p *provider) driveGetPermission(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	permissionID := strings.TrimSpace(asString(input["permissionId"]))
	if fileID == "" || permissionID == "" {
		return map[string]any{"service": "drive", "operation": "getPermission", "error_code": "invalid_argument", "message": "missing fileId or permissionId"}, errMissingFileOrPermissionID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "permission", fileID, permissionID)
	return p.runCLI(cleanArgs(args), "drive", "getPermission")
}

func (p *provider) driveListFiles(ctx context.Context, input map[string]any) (map[string]any, error) {
	driveListSearchNormalizeInput(input)
	query := strings.TrimSpace(asString(input["query"]))
	global := asBool(input["global"])
	parentID := strings.TrimSpace(asString(input["parentId"]))
	if global && parentID != "" {
		return map[string]any{"service": "drive", "operation": "listFiles", "error_code": "invalid_argument", "message": "global cannot be combined with parentId"}, errors.New("global cannot be combined with parentId")
	}
	if parentID == "" {
		parentID = "root"
	}
	// Redirect to folders-only when: no query (agent often means "list folders" when using {}),
	// or query is trashed=false / folder mimeType — so response has only folders and fits gateway.
	var redirectToFoldersOnly bool
	if !global && query == "" {
		redirectToFoldersOnly = true
	} else {
		qLower := strings.ToLower(query)
		redirectToFoldersOnly = strings.Contains(qLower, "application/vnd.google-apps.folder") ||
			(strings.Contains(qLower, "mimetype") && strings.Contains(qLower, "folder")) ||
			qLower == "trashed=false"
	}
	if redirectToFoldersOnly {
		searchInput := make(map[string]any)
		for k, v := range input {
			searchInput[k] = v
		}
		searchInput["query"] = "mimeType = 'application/vnd.google-apps.folder' and '" + parentID + "' in parents"
		searchInput["rawQuery"] = true
		return p.driveSearchFiles(ctx, searchInput)
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "ls")
	if global {
		args = append(args, "--global")
	}
	if parent := strings.TrimSpace(asString(input["parentId"])); parent != "" {
		args = append(args, "--parent", parent)
	}
	if query := strings.TrimSpace(asString(input["query"])); query != "" {
		args = append(args, "--query", query)
	}
	if nMax, ok := asInt(input["max"]); ok && nMax > 0 {
		args = append(args, "--max", strconv.FormatInt(nMax, 10))
	}
	pageVal := strings.TrimSpace(asString(input["page"]))
	if pageVal != "" && !strings.EqualFold(pageVal, "null") {
		args = append(args, "--page", pageVal)
	} else {
		// Paginated mode (no --all): one page of mcpDrivePageSize so response fits gateway/exec limit.
		if _, hasMax := input["max"]; !hasMax {
			args = append(args, "--max", strconv.Itoa(mcpDrivePageSize))
		}
		args = append(args, "--compact")
	}
	if _, ok := input["allDrives"]; ok {
		if asBool(input["allDrives"]) {
			args = append(args, "--all-drives")
		} else {
			args = append(args, "--no-all-drives")
		}
	}
	return p.runCLI(cleanArgs(args), "drive", "listFiles")
}

func (p *provider) driveSearchFiles(_ context.Context, input map[string]any) (map[string]any, error) {
	driveListSearchNormalizeInput(input)
	query := strings.TrimSpace(asString(input["query"]))
	if query == "" {
		return map[string]any{"service": "drive", "operation": "searchFiles", "error_code": "invalid_argument", "message": "missing query"}, errMissingQuery
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "search")
	if asBool(input["rawQuery"]) {
		args = append(args, "--raw-query")
	}
	if nMax, ok := asInt(input["max"]); ok && nMax > 0 {
		args = append(args, "--max", strconv.FormatInt(nMax, 10))
	}
	pageVal := strings.TrimSpace(asString(input["page"]))
	if pageVal != "" && !strings.EqualFold(pageVal, "null") {
		args = append(args, "--page", pageVal)
	} else {
		// Paginated mode (no --all): one page of mcpDrivePageSize so response fits gateway/exec limit.
		if _, hasMax := input["max"]; !hasMax {
			args = append(args, "--max", strconv.Itoa(mcpDrivePageSize))
		}
		args = append(args, "--compact")
	}
	// Default allDrives to true so search includes shared drives; only restrict when explicitly false
	if v, ok := input["allDrives"]; ok && !asBool(v) {
		args = append(args, "--no-all-drives")
	} else {
		args = append(args, "--all-drives")
	}
	args = append(args, query)
	return p.runCLI(cleanArgs(args), "drive", "searchFiles")
}

func (p *provider) driveGetFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "getFile", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "get", fileID)
	return p.runCLI(cleanArgs(args), "drive", "getFile")
}

func (p *provider) driveUploadFile(_ context.Context, input map[string]any) (map[string]any, error) {
	localPath := strings.TrimSpace(asString(input["localPath"]))
	if localPath == "" {
		return map[string]any{"service": "drive", "operation": "uploadFile", "error_code": "invalid_argument", "message": "missing localPath"}, errMissingLocalPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "upload", localPath)
	if v := strings.TrimSpace(asString(input["name"])); v != "" {
		args = append(args, "--name", v)
	}
	if v := strings.TrimSpace(asString(input["parentId"])); v != "" {
		args = append(args, "--parent", v)
	}
	if v := strings.TrimSpace(asString(input["replaceFileId"])); v != "" {
		args = append(args, "--replace", v)
	}
	if v := strings.TrimSpace(asString(input["mimeType"])); v != "" {
		args = append(args, "--mime-type", v)
	}
	if asBool(input["convert"]) {
		args = append(args, "--convert")
	}
	if v := strings.TrimSpace(asString(input["convertTo"])); v != "" {
		args = append(args, "--convert-to", v)
	}
	if asBool(input["keepRevisionForever"]) {
		args = append(args, "--keep-revision-forever")
	}
	return p.runCLI(cleanArgs(args), "drive", "uploadFile")
}

func (p *provider) driveDownloadFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "downloadFile", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "download", fileID)
	if out := strings.TrimSpace(asString(input["out"])); out != "" {
		args = append(args, "--out", out)
	}
	if format := strings.TrimSpace(asString(input["format"])); format != "" {
		args = append(args, "--format", format)
	}
	return p.runCLI(cleanArgs(args), "drive", "downloadFile")
}

func (p *provider) driveListPermissions(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "listPermissions", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "permissions", fileID)
	if nMax, ok := asInt(input["max"]); ok && nMax > 0 {
		args = append(args, "--max", strconv.FormatInt(nMax, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
	}
	return p.runCLI(cleanArgs(args), "drive", "listPermissions")
}

func (p *provider) driveListComments(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "listComments", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "comments", "list", fileID)
	if nMax, ok := asInt(input["max"]); ok && nMax > 0 {
		args = append(args, "--max", strconv.FormatInt(nMax, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
	}
	if asBool(input["all"]) {
		args = append(args, "--all")
	}
	if asBool(input["includeQuoted"]) {
		args = append(args, "--include-quoted")
	}
	if asBool(input["failEmpty"]) {
		args = append(args, "--fail-empty")
	}
	return p.runCLI(cleanArgs(args), "drive", "listComments")
}

func (p *provider) driveDeleteFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "deleteFile", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	if asBool(input["validateOnly"]) {
		planned := map[string]any{"fileId": fileID}
		if asBool(input["permanent"]) {
			planned["permanent"] = true
		}
		return map[string]any{
			"service":      "drive",
			"operation":    "deleteFile",
			"validateOnly": true,
			"planned":      planned,
		}, nil
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "delete", fileID)
	if asBool(input["permanent"]) {
		args = append(args, "--permanent")
	}
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	return p.runCLI(cleanArgs(args), "drive", "deleteFile")
}

func (p *provider) driveMoveFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	parentID := strings.TrimSpace(asString(input["parentId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "moveFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if parentID == "" {
		return map[string]any{"service": "drive", "operation": "moveFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing parentId"}, errMissingPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "move", fileID, "--parent", parentID)
	return p.runCLI(cleanArgs(args), "drive", "moveFile")
}

func (p *provider) driveRenameFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	name := strings.TrimSpace(asString(input["name"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "renameFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if name == "" {
		return map[string]any{"service": "drive", "operation": "renameFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing name"}, errors.New("missing name")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "rename", fileID, name)
	return p.runCLI(cleanArgs(args), "drive", "renameFile")
}

func (p *provider) driveShareFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	to := strings.TrimSpace(asString(input["to"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if to == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing to (anyone|user|domain)"}, errors.New("missing to")
	}
	to = strings.ToLower(to)
	if to != "anyone" && to != "user" && to != "domain" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "to must be anyone|user|domain"}, errors.New("invalid to")
	}
	if to == "user" && strings.TrimSpace(asString(input["email"])) == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "email required when to=user"}, errors.New("missing email")
	}
	if to == "domain" && strings.TrimSpace(asString(input["domain"])) == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "domain required when to=domain"}, errors.New("missing domain")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "share", fileID, "--to", to)
	if email := strings.TrimSpace(asString(input["email"])); email != "" {
		args = append(args, "--email", email)
	}
	if domain := strings.TrimSpace(asString(input["domain"])); domain != "" {
		args = append(args, "--domain", domain)
	}
	if role := strings.TrimSpace(asString(input["role"])); role != "" {
		args = append(args, "--role", role)
	}
	if asBool(input["discoverable"]) {
		args = append(args, "--discoverable")
	}
	return p.runCLI(cleanArgs(args), "drive", "shareFile")
}

func (p *provider) driveUnshare(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	permID := strings.TrimSpace(asString(input["permissionId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "unshare", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if permID == "" {
		return map[string]any{"service": "drive", "operation": "unshare", "error_code": server.ErrorCodeInvalidArgument, "message": "missing permissionId"}, errMissingFileOrPermissionID
	}
	if asBool(input["validateOnly"]) {
		return map[string]any{
			"service":      "drive",
			"operation":    "unshare",
			"validateOnly": true,
			"planned":      map[string]any{"fileId": fileID, "permissionId": permID},
		}, nil
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "unshare", fileID, permID)
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	return p.runCLI(cleanArgs(args), "drive", "unshare")
}

func (p *provider) driveCreateComment(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	content := strings.TrimSpace(asString(input["content"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "createComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if content == "" {
		return map[string]any{"service": "drive", "operation": "createComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing content"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "comments", "create", fileID, content)
	if quoted := strings.TrimSpace(asString(input["quoted"])); quoted != "" {
		args = append(args, "--quoted", quoted)
	}
	return p.runCLI(cleanArgs(args), "drive", "createComment")
}

func (p *provider) driveDeleteComment(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	commentID := strings.TrimSpace(asString(input["commentId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "deleteComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if commentID == "" {
		return map[string]any{"service": "drive", "operation": "deleteComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing commentId"}, errMissingFileOrPermissionID
	}
	if asBool(input["validateOnly"]) {
		return map[string]any{
			"service":      "drive",
			"operation":    "deleteComment",
			"validateOnly": true,
			"planned":      map[string]any{"fileId": fileID, "commentId": commentID},
		}, nil
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "comments", "delete", fileID, commentID)
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	return p.runCLI(cleanArgs(args), "drive", "deleteComment")
}

func (p *provider) driveCopyFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	name := strings.TrimSpace(asString(input["name"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "copyFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if name == "" {
		return map[string]any{"service": "drive", "operation": "copyFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing name"}, errors.New("missing name")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "copy", fileID, name)
	if parentID := strings.TrimSpace(asString(input["parentId"])); parentID != "" {
		args = append(args, "--parent", parentID)
	}
	return p.runCLI(cleanArgs(args), "drive", "copyFile")
}

const driveBulkMaxOps = 50

func (p *provider) driveBulkExecute(_ context.Context, input map[string]any) (map[string]any, error) {
	raw, ok := input["operations"].([]any)
	if !ok || len(raw) == 0 {
		return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty operations array"}, errMissingRequest
	}
	if len(raw) > driveBulkMaxOps {
		return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "operations exceeds max " + strconv.Itoa(driveBulkMaxOps)}, errors.New("operations exceeds max")
	}
	operations := make([]map[string]any, 0, len(raw))
	for i, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "operations[" + strconv.Itoa(i) + "] must be an object"}, errMissingRequest
		}
		if strings.TrimSpace(asString(m["fileId"])) == "" {
			return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "operations[" + strconv.Itoa(i) + "] missing fileId"}, errMissingFileID
		}
		operations = append(operations, m)
	}
	if asBool(input["validateOnly"]) {
		planned := append([]map[string]any(nil), operations...)
		return map[string]any{
			"service":      "drive",
			"operation":    "bulkExecute",
			"validateOnly": true,
			"planned":      planned,
			"count":        len(planned),
		}, nil
	}
	if p == nil || p.exec == nil {
		return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInternal, "message": "executor not configured"}, errExecutorNotConfigured
	}
	baseArgs := make([]string, 0, 12)
	baseArgs = append(baseArgs, "--json")
	baseArgs = append(baseArgs, policyArgs(input)...)
	baseArgs = append(baseArgs, maybeAccountArgs(input)...)
	var results []map[string]any
	var succeeded, failed int
	for _, m := range operations {
		op := strings.ToLower(strings.TrimSpace(asString(m["op"])))
		fileID := strings.TrimSpace(asString(m["fileId"]))
		var args []string
		switch op {
		case "move":
			parentID := strings.TrimSpace(asString(m["parentId"]))
			if parentID == "" {
				results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "missing parentId"})
				failed++
				continue
			}
			args = append(append([]string{}, baseArgs...), "drive", "move", fileID, "--parent", parentID)
		case "rename":
			name := strings.TrimSpace(asString(m["name"]))
			if name == "" {
				results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "missing name"})
				failed++
				continue
			}
			args = append(append([]string{}, baseArgs...), "drive", "rename", fileID, name)
		case "share":
			to := strings.TrimSpace(asString(m["to"]))
			if to == "" {
				results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "missing to"})
				failed++
				continue
			}
			args = append(append([]string{}, baseArgs...), "drive", "share", fileID, "--to", to)
			if email := asString(m["email"]); email != "" {
				args = append(args, "--email", email)
			}
			if domain := asString(m["domain"]); domain != "" {
				args = append(args, "--domain", domain)
			}
			if role := asString(m["role"]); role != "" {
				args = append(args, "--role", role)
			}
		case "delete":
			args = append(append([]string{}, baseArgs...), "drive", "delete", fileID)
			if asBool(m["permanent"]) {
				args = append(args, "--permanent")
			}
			args = append(args, "--force")
		default:
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "unsupported op"})
			failed++
			continue
		}
		stdout, stderr, err := p.exec(cleanArgs(args))
		if err != nil || strings.TrimSpace(stderr) != "" {
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": strings.TrimSpace(stderr), "execErr": err != nil})
			failed++
			continue
		}
		var parsed map[string]any
		if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); jsonErr == nil {
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": true, "result": parsed})
		} else {
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": true})
		}
		succeeded++
	}
	return map[string]any{
		"service":   "drive",
		"operation": "bulkExecute",
		"results":   results,
		"succeeded": succeeded,
		"failed":    failed,
	}, nil
}

func (p *provider) gmailSearch(_ context.Context, input map[string]any) (map[string]any, error) {
	query := strings.TrimSpace(asString(input["query"]))
	if query == "" {
		return map[string]any{"service": "gmail", "operation": "search", "error_code": server.ErrorCodeInvalidArgument, "message": "missing query"}, errMissingQuery
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "gmail", "search", query)
	if nMax, ok := asInt(input["max"]); ok && nMax > 0 {
		args = append(args, "--max", strconv.FormatInt(nMax, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
	}
	return p.runCLI(cleanArgs(args), "gmail", "search")
}

func (p *provider) gmailSend(_ context.Context, input map[string]any) (map[string]any, error) {
	to := strings.TrimSpace(asString(input["to"]))
	subject := strings.TrimSpace(asString(input["subject"]))
	body := strings.TrimSpace(asString(input["body"]))
	bodyHTML := strings.TrimSpace(asString(input["bodyHtml"]))
	if to == "" {
		return map[string]any{"service": "gmail", "operation": "send", "error_code": server.ErrorCodeInvalidArgument, "message": "missing to"}, errMissingQuery
	}
	if subject == "" {
		return map[string]any{"service": "gmail", "operation": "send", "error_code": server.ErrorCodeInvalidArgument, "message": "missing subject"}, errMissingQuery
	}
	if body == "" && bodyHTML == "" {
		return map[string]any{"service": "gmail", "operation": "send", "error_code": server.ErrorCodeInvalidArgument, "message": "missing body or bodyHtml"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "gmail", "send", "--to", to, "--subject", subject)
	if bodyHTML != "" {
		args = append(args, "--body-html", bodyHTML)
	} else {
		args = append(args, "--body", body)
	}
	if cc := strings.TrimSpace(asString(input["cc"])); cc != "" {
		args = append(args, "--cc", cc)
	}
	if bcc := strings.TrimSpace(asString(input["bcc"])); bcc != "" {
		args = append(args, "--bcc", bcc)
	}
	if from := strings.TrimSpace(asString(input["from"])); from != "" {
		args = append(args, "--from", from)
	}
	return p.runCLI(cleanArgs(args), "gmail", "send")
}

func (p *provider) calendarEvents(_ context.Context, input map[string]any) (map[string]any, error) {
	from := strings.TrimSpace(asString(input["from"]))
	to := strings.TrimSpace(asString(input["to"]))
	if from == "" {
		return map[string]any{"service": "calendar", "operation": "events", "error_code": server.ErrorCodeInvalidArgument, "message": "missing from"}, errMissingQuery
	}
	if to == "" {
		return map[string]any{"service": "calendar", "operation": "events", "error_code": server.ErrorCodeInvalidArgument, "message": "missing to"}, errMissingQuery
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	calendarID := strings.TrimSpace(asString(input["calendarId"]))
	if calendarID != "" {
		args = append(args, "calendar", "events", calendarID, "--from", from, "--to", to)
	} else {
		args = append(args, "calendar", "events", "--from", from, "--to", to)
	}
	if nMax, ok := asInt(input["max"]); ok && nMax > 0 {
		args = append(args, "--max", strconv.FormatInt(nMax, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
	}
	if q := strings.TrimSpace(asString(input["query"])); q != "" {
		args = append(args, "--query", q)
	}
	return p.runCLI(cleanArgs(args), "calendar", "events")
}

func (p *provider) contactsList(_ context.Context, input map[string]any) (map[string]any, error) {
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "contacts", "list")
	if nMax, ok := asInt(input["max"]); ok && nMax > 0 {
		args = append(args, "--max", strconv.FormatInt(nMax, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
	}
	return p.runCLI(cleanArgs(args), "contacts", "list")
}

func (p *provider) runCLI(args []string, service, operation string) (map[string]any, error) {
	if p == nil || p.exec == nil {
		return map[string]any{
			"service":    service,
			"operation":  operation,
			"error_code": server.ErrorCodeInternal,
			"message":    "mcp executor is not configured",
		}, errExecutorNotConfigured
	}

	stdout, stderr, execErr := p.exec(args)
	if execErr != nil && strings.TrimSpace(stderr) == "" {
		return map[string]any{
			"service":    service,
			"operation":  operation,
			"error_code": server.ErrorCodeAPI,
			"message":    execErr.Error(),
		}, execErr
	}
	if strings.TrimSpace(stderr) != "" {
		var parsed map[string]any
		if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &parsed); jsonErr == nil {
			if errObj, ok := parsed["error"].(map[string]any); ok {
				errObj["service"] = service
				errObj["operation"] = operation
				return errObj, errToolCommandFailed
			}
		}
		return map[string]any{"service": service, "operation": operation, "error_code": server.ErrorCodeAPI, "message": strings.TrimSpace(stderr)}, errToolStderr
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		return map[string]any{"service": service, "operation": operation, "error_code": server.ErrorCodeInvalidJSON, "message": "failed to parse command output"}, fmt.Errorf("parse command output: %w", err)
	}
	parsed["service"] = service
	parsed["operation"] = operation
	return parsed, nil
}
