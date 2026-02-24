package googleapi

import "testing"

func TestConfigureRetryPolicy(t *testing.T) {
	orig := runtimeRetryConfig

	t.Cleanup(func() { runtimeRetryConfig = orig })

	if err := ConfigureRetryPolicy(4, "750ms"); err != nil {
		t.Fatalf("ConfigureRetryPolicy: %v", err)
	}

	if runtimeRetryConfig.MaxRetries429 != 4 || runtimeRetryConfig.MaxRetries5xx != 4 {
		t.Fatalf("unexpected retries config: %#v", runtimeRetryConfig)
	}

	if got := runtimeRetryConfig.BaseDelay.String(); got != "750ms" {
		t.Fatalf("base delay=%s", got)
	}
}

func TestConfigureRetryPolicy_InvalidBackoff(t *testing.T) {
	if err := ConfigureRetryPolicy(1, "notaduration"); err == nil {
		t.Fatal("expected error for invalid backoff")
	}

	if err := ConfigureRetryPolicy(1, "0s"); err == nil {
		t.Fatal("expected error for non-positive backoff")
	}
}
