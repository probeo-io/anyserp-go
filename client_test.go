package anyserp

import (
	"context"
	"os"
	"testing"
)

func TestNewWithConfig(t *testing.T) {
	client := New(&Config{
		Serper:  &ProviderConfig{APIKey: "test-serper-key"},
		SerpAPI: &ProviderConfig{APIKey: "test-serpapi-key"},
		Bing:    &ProviderConfig{APIKey: "test-bing-key"},
	})

	providers := client.Providers()
	if len(providers) != 3 {
		t.Fatalf("expected 3 providers, got %d: %v", len(providers), providers)
	}

	expected := map[string]bool{"serper": true, "serpapi": true, "bing": true}
	for _, p := range providers {
		if !expected[p] {
			t.Errorf("unexpected provider: %s", p)
		}
	}
}

func TestNewWithEnvVars(t *testing.T) {
	os.Setenv("SERPER_API_KEY", "env-serper-key")
	os.Setenv("BRAVE_API_KEY", "env-brave-key")
	defer os.Unsetenv("SERPER_API_KEY")
	defer os.Unsetenv("BRAVE_API_KEY")

	client := New(nil)

	providers := client.Providers()
	if len(providers) < 2 {
		t.Fatalf("expected at least 2 providers, got %d: %v", len(providers), providers)
	}

	found := map[string]bool{}
	for _, p := range providers {
		found[p] = true
	}
	if !found["serper"] {
		t.Error("expected serper provider from env var")
	}
	if !found["brave"] {
		t.Error("expected brave provider from env var")
	}
}

func TestNewWithGoogleCSE(t *testing.T) {
	// Google CSE requires both API key and engine ID
	client := New(&Config{
		Google: &ProviderConfig{APIKey: "google-key"},
	})
	providers := client.Providers()
	for _, p := range providers {
		if p == "google" {
			t.Error("google should not be registered without engine ID")
		}
	}

	client = New(&Config{
		Google: &ProviderConfig{APIKey: "google-key", EngineID: "engine-id"},
	})
	found := false
	for _, p := range client.Providers() {
		if p == "google" {
			found = true
		}
	}
	if !found {
		t.Error("expected google provider when both key and engine ID are provided")
	}
}

func TestNewWithDataForSEO(t *testing.T) {
	client := New(&Config{
		DataForSEO: &DataForSeoConfig{Login: "login", Password: "pass"},
	})
	found := false
	for _, p := range client.Providers() {
		if p == "dataforseo" {
			found = true
		}
	}
	if !found {
		t.Error("expected dataforseo provider")
	}
}

func TestProviderPrefixRouting(t *testing.T) {
	// Create a client with a mock adapter
	client := New(&Config{})
	mock := &mockAdapter{name: "mock1", results: &SearchResponse{Provider: "mock1", Query: "test"}}
	client.registry.Register("mock1", mock)

	resp, err := client.Search(context.Background(), SearchRequest{Query: "mock1/test query"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "mock1" {
		t.Errorf("expected provider mock1, got %s", resp.Provider)
	}
	if mock.lastQuery != "test query" {
		t.Errorf("expected query 'test query', got '%s'", mock.lastQuery)
	}
}

func TestAliasRouting(t *testing.T) {
	client := New(&Config{
		Aliases: map[string]string{"g": "mock1"},
	})
	mock := &mockAdapter{name: "mock1", results: &SearchResponse{Provider: "mock1", Query: "test"}}
	client.registry.Register("mock1", mock)

	resp, err := client.Search(context.Background(), SearchRequest{Query: "g/test query"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "mock1" {
		t.Errorf("expected provider mock1, got %s", resp.Provider)
	}
}

func TestDefaultsApplication(t *testing.T) {
	client := New(&Config{
		Defaults: &SearchDefaults{
			Num:      20,
			Country:  "us",
			Language: "en",
			Safe:     true,
		},
	})
	mock := &mockAdapter{name: "mock1", results: &SearchResponse{Provider: "mock1"}}
	client.registry.Register("mock1", mock)

	_, err := client.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastRequest.Num != 20 {
		t.Errorf("expected num 20, got %d", mock.lastRequest.Num)
	}
	if mock.lastRequest.Country != "us" {
		t.Errorf("expected country us, got %s", mock.lastRequest.Country)
	}
	if mock.lastRequest.Language != "en" {
		t.Errorf("expected language en, got %s", mock.lastRequest.Language)
	}
	if !mock.lastRequest.Safe {
		t.Error("expected safe to be true")
	}
}

func TestDefaultsNotOverrideExplicit(t *testing.T) {
	client := New(&Config{
		Defaults: &SearchDefaults{
			Num:     20,
			Country: "us",
		},
	})
	mock := &mockAdapter{name: "mock1", results: &SearchResponse{Provider: "mock1"}}
	client.registry.Register("mock1", mock)

	_, err := client.Search(context.Background(), SearchRequest{
		Query:   "test",
		Num:     5,
		Country: "gb",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastRequest.Num != 5 {
		t.Errorf("expected num 5, got %d", mock.lastRequest.Num)
	}
	if mock.lastRequest.Country != "gb" {
		t.Errorf("expected country gb, got %s", mock.lastRequest.Country)
	}
}

func TestSearchNoProviders(t *testing.T) {
	client := New(&Config{})
	_, err := client.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error when no providers configured")
	}
	if e, ok := err.(*AnySerpError); ok {
		if e.Code != 400 {
			t.Errorf("expected error code 400, got %d", e.Code)
		}
	} else {
		t.Errorf("expected AnySerpError, got %T", err)
	}
}

func TestSearchWithFallback(t *testing.T) {
	client := New(&Config{})
	failing := &mockAdapter{name: "failing", err: NewAnySerpError(500, "fail", nil)}
	success := &mockAdapter{name: "success", results: &SearchResponse{Provider: "success", Query: "test"}}
	client.registry.Register("failing", failing)
	client.registry.Register("success", success)

	resp, err := client.SearchWithFallback(context.Background(), SearchRequest{Query: "test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "success" {
		t.Errorf("expected provider success, got %s", resp.Provider)
	}
}

func TestSearchWithFallbackAllFail(t *testing.T) {
	client := New(&Config{})
	failing1 := &mockAdapter{name: "fail1", err: NewAnySerpError(500, "fail1", nil)}
	failing2 := &mockAdapter{name: "fail2", err: NewAnySerpError(502, "fail2", nil)}
	client.registry.Register("fail1", failing1)
	client.registry.Register("fail2", failing2)

	_, err := client.SearchWithFallback(context.Background(), SearchRequest{Query: "test"}, nil)
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

func TestVersion(t *testing.T) {
	if Version != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %s", Version)
	}
}

// mockAdapter is a test helper implementing SearchAdapter.
type mockAdapter struct {
	name        string
	results     *SearchResponse
	err         error
	lastQuery   string
	lastRequest SearchRequest
}

func (m *mockAdapter) Name() string { return m.name }

func (m *mockAdapter) SupportsType(_ SearchType) bool { return true }

func (m *mockAdapter) Search(_ context.Context, request SearchRequest) (*SearchResponse, error) {
	m.lastQuery = request.Query
	m.lastRequest = request
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}
