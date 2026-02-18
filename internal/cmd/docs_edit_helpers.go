package cmd

import (
	"reflect"

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

// docsRequestOperationCount returns the number of operation fields set in a docs.Request.
func docsRequestOperationCount(r *docs.Request) int {
	if r == nil {
		return 0
	}
	v := reflect.ValueOf(*r)
	t := reflect.TypeOf(*r)
	count := 0
	for i := range t.NumField() {
		name := t.Field(i).Name
		if name == "ForceSendFields" || name == "NullFields" || name == "ServerResponse" {
			continue
		}
		fv := v.Field(i)
		switch fv.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			if !fv.IsNil() {
				count++
			}
		}
	}
	return count
}

// docsRequestOperationName returns the name of the first set operation field in a docs.Request.
func docsRequestOperationName(r *docs.Request) string {
	if r == nil {
		return ""
	}
	v := reflect.ValueOf(*r)
	t := reflect.TypeOf(*r)
	for i := range t.NumField() {
		name := t.Field(i).Name
		if name == "ForceSendFields" || name == "NullFields" || name == "ServerResponse" {
			continue
		}
		fv := v.Field(i)
		switch fv.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			if !fv.IsNil() {
				return name
			}
		}
	}
	return ""
}
