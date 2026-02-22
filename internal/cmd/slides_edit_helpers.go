package cmd

// newSlidesEditError creates a structured edit error scoped to the Slides service.
func newSlidesEditError(op, presentationID, code, msg string, cause error) error {
	return NewEditError("slides", op, presentationID, code, msg, cause)
}
