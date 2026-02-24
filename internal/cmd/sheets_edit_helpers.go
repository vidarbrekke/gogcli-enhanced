package cmd

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
