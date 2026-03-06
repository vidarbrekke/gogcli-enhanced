package cmd

import (
	"errors"
	"testing"

	"github.com/steipete/gogcli/internal/parity/normalize"
)

func TestBackendError_Error(t *testing.T) {
	e := &BackendError{Env: &normalize.CanonicalEnvelope{ErrorCode: "not_found", HTTPStatus: 404}}
	if e.Error() == "" {
		t.Error("Error() should not be empty")
	}
}

func TestBackendError_JSONErrorFields(t *testing.T) {
	env := &normalize.CanonicalEnvelope{
		ErrorCode:    "unauthenticated",
		HTTPStatus:   401,
		Service:      "gmail",
		Operation:    "labels list",
		GoogleReason: "authError",
	}
	e := &BackendError{Env: env}
	fields := e.JSONErrorFields()
	if fields["error_code"] != "unauthenticated" {
		t.Errorf("error_code = %v", fields["error_code"])
	}
	if fields["http_status"] != 401 {
		t.Errorf("http_status = %v", fields["http_status"])
	}
	if fields["service"] != "gmail" {
		t.Errorf("service = %v", fields["service"])
	}
	if fields["google_reason"] != "authError" {
		t.Errorf("google_reason = %v", fields["google_reason"])
	}
}

func TestBackendError_stableExitCode(t *testing.T) {
	tests := []struct {
		errorCode   string
		wantCode    int
		wantWrapped bool // when false, stableExitCode returns err as-is (code 1)
	}{
		{"unauthenticated", exitCodeAuthRequired, true},
		{"not_found", exitCodeNotFound, true},
		{"permission_denied", exitCodePermissionDenied, true},
		{"resource_exhausted", exitCodeRateLimited, true},
		{"invalid_argument", 2, true},
		{"unknown", 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.errorCode, func(t *testing.T) {
			err := &BackendError{Env: &normalize.CanonicalEnvelope{ErrorCode: tt.errorCode, HTTPStatus: 401}}
			wrapped := stableExitCode(err)
			if !tt.wantWrapped {
				if !errors.Is(wrapped, err) {
					t.Errorf("stableExitCode should return same error for code 1, got %T", wrapped)
				}
				return
			}
			var ee *ExitError
			if !errors.As(wrapped, &ee) {
				t.Fatalf("stableExitCode did not return ExitError")
			}
			if ee.Code != tt.wantCode {
				t.Errorf("ExitError.Code = %d, want %d", ee.Code, tt.wantCode)
			}
		})
	}
}
