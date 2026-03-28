package anyserp

import (
	"testing"
)

func TestAnySerpErrorWithProvider(t *testing.T) {
	err := NewAnySerpError(401, "Unauthorized", map[string]interface{}{
		"provider_name": "serper",
	})
	expected := "anyserp [serper] 401: Unauthorized"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAnySerpErrorWithoutProvider(t *testing.T) {
	err := NewAnySerpError(500, "Internal error", nil)
	expected := "anyserp 500: Internal error"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAnySerpErrorNilMetadata(t *testing.T) {
	err := NewAnySerpError(400, "Bad request", nil)
	if err.Metadata == nil {
		t.Fatal("expected non-nil metadata map")
	}
	if len(err.Metadata) != 0 {
		t.Errorf("expected empty metadata, got %v", err.Metadata)
	}
}

func TestAnySerpErrorFields(t *testing.T) {
	meta := map[string]interface{}{"raw": "data"}
	err := NewAnySerpError(429, "Rate limited", meta)
	if err.Code != 429 {
		t.Errorf("expected code 429, got %d", err.Code)
	}
	if err.Message != "Rate limited" {
		t.Errorf("expected message 'Rate limited', got %q", err.Message)
	}
	if err.Metadata["raw"] != "data" {
		t.Errorf("expected metadata raw=data, got %v", err.Metadata["raw"])
	}
}
