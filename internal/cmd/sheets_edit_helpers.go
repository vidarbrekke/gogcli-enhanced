package cmd

import (
	"reflect"

	"google.golang.org/api/sheets/v4"
)

// SheetsEditSafetyFlags is the shared agentic safety flags for Sheets edit commands.
type SheetsEditSafetyFlags = AgenticEditSafetyFlags

// newSheetsEditError creates a structured edit error scoped to the Sheets service.
func newSheetsEditError(op, spreadsheetID, code, msg string, cause error) error {
	return NewEditError("sheets", op, spreadsheetID, code, msg, cause)
}

// isSheetsNotFound checks if an error is a 404 from the Sheets API.
func isSheetsNotFound(err error) bool {
	return IsNotFound(err)
}

// sheetsRequestOperationCount returns the number of operation fields set in a sheets.Request.
func sheetsRequestOperationCount(r *sheets.Request) int {
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

// sheetsRequestOperationName returns the name of the first set operation field in a sheets.Request.
func sheetsRequestOperationName(r *sheets.Request) string {
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
