package cmd

import (
	"os"
	"testing"
)

func TestBackend_defaultNative(t *testing.T) {
	os.Unsetenv("GOG_BACKEND")
	defer os.Unsetenv("GOG_BACKEND")
	if got := Backend(); got != BackendNative {
		t.Errorf("Backend() = %q, want %q", got, BackendNative)
	}
}

func TestBackend_gwsWhenEnvSet(t *testing.T) {
	os.Setenv("GOG_BACKEND", "gws")
	defer os.Unsetenv("GOG_BACKEND")
	if got := Backend(); got != BackendGWS {
		t.Errorf("Backend() = %q, want %q", got, BackendGWS)
	}
}

func TestBackend_nativeWhenEnvEmptyOrOther(t *testing.T) {
	for _, v := range []string{"", "native", "NATIVE", "other"} {
		t.Run(v, func(t *testing.T) {
			if v == "" {
				os.Unsetenv("GOG_BACKEND")
			} else {
				os.Setenv("GOG_BACKEND", v)
			}
			defer os.Unsetenv("GOG_BACKEND")
			if got := Backend(); got != BackendNative {
				t.Errorf("Backend() = %q, want %q", got, BackendNative)
			}
		})
	}
}

func TestBackend_gwsCaseInsensitive(t *testing.T) {
	for _, v := range []string{"gws", "GWS", "Gws"} {
		t.Run(v, func(t *testing.T) {
			os.Setenv("GOG_BACKEND", v)
			defer os.Unsetenv("GOG_BACKEND")
			if got := Backend(); got != BackendGWS {
				t.Errorf("Backend() = %q, want %q", got, BackendGWS)
			}
		})
	}
}
