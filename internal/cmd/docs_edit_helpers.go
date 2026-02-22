package cmd

import (
	"google.golang.org/api/docs/v1"
)

// newDocsEditError creates a structured edit error scoped to the Docs service.
func newDocsEditError(op, docID, code, msg string, cause error) error {
	return NewEditError("docs", op, docID, code, msg, cause)
}

// isDocsNotFound checks if an error is a 404 from the Docs API.
func isDocsNotFound(err error) bool {
	return IsNotFound(err)
}

func docsAppendIndex(doc *docs.Document) int64 {
	if doc == nil || doc.Body == nil || len(doc.Body.Content) == 0 {
		return 1
	}
	last := doc.Body.Content[len(doc.Body.Content)-1]
	if last == nil || last.EndIndex <= 1 {
		return 1
	}
	return last.EndIndex - 1
}

