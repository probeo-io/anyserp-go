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
	client := New(&Config{
		Google: &ProviderConfig{APIKey: "google-key"},
	})
	for _, p := range client.Providers() {
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
		t.Error("expected google provider when both key and engine ID provided")
	}
}

func TestNewWithGoogleCSEEnvVars(t *testing.T) {
	os.Setenv("GOOGLE_CSE_API_KEY", "env-google-key")
	os.Setenv("GOOGLE_CSE_ENGINE_ID", "env-engine-id")
	defer os.Unsetenv("GOOGLE_CSE_API_KEY")
	defer os.Unsetenv("GOOGLE_CSE_ENGINE_ID")

	client := New(nil)
	found := false
	for _, p := range client.Providers() {
		if p == "google" {
			found = true
		}
	}
	if !found {
		t.Error("expected google from env vars")
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

func TestNewWithDataForSEOEnvVars(t *testing.T) {
	os.Setenv("DATAFORSEO_LOGIN", "env-login")
	os.Setenv("DATAFORSEO_PASSWORD", "env-pass")
	defer os.Unsetenv("DATAFORSEO_LOGIN")
	defer os.Unsetenv("DATAFORSEO_PASSWORD")

	client := New(nil)
	found := false
	for _, p := range client.Providers() {
		if p == "dataforseo" {
			found = true
		}
	}
	if !found {
		t.Error("expected dataforseo from env vars")
	}
}

func TestNewWithAllProviders(t *testing.T) {
	client := New(&Config{
		Serper:      &ProviderConfig{APIKey: "k"},
		SerpAPI:     &ProviderConfig{APIKey: "k"},
		Google:      &ProviderConfig{APIKey: "k", EngineID: "cx"},
		Bing:        &ProviderConfig{APIKey: "k"},
		Brave:       &ProviderConfig{APIKey: "k"},
		DataForSEO:  &DataForSeoConfig{Login: "l", Password: "p"},
		SearchAPI:   &ProviderConfig{APIKey: "k"},
		ValueSERP:   &ProviderConfig{APIKey: "k"},
		ScrapingDog: &ProviderConfig{APIKey: "k"},
		BrightData:  &ProviderConfig{APIKey: "k"},
		SearchCans:  &ProviderConfig{APIKey: "k"},
	})

	if len(client.Providers()) != 11 {
		t.Errorf("expected 11 providers, got %d: %v", len(client.Providers()), client.Providers())
	}
}

func TestNewNilConfig(t *testing.T) {
	client := New(nil)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestProviderPrefixRouting(t *testing.T) {
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

func TestProviderPrefixInvalidProvider(t *testing.T) {
	client := New(&Config{})
	mock := &mockAdapter{name: "mock1", results: &SearchResponse{Provider: "mock1"}}
	client.registry.Register("mock1", mock)

	// "notreal/" should not match since "notreal" is not registered
	// So it should use mock1 as the first available provider
	resp, err := client.Search(context.Background(), SearchRequest{Query: "notreal/query"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The query should remain "notreal/query" since "notreal" is not a provider
	if mock.lastQuery != "notreal/query" {
		t.Errorf("expected query 'notreal/query', got '%s'", mock.lastQuery)
	}
	if resp.Provider != "mock1" {
		t.Errorf("expected provider mock1, got %s", resp.Provider)
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

func TestDefaultsNilDefaults(t *testing.T) {
	client := New(&Config{})
	mock := &mockAdapter{name: "mock1", results: &SearchResponse{Provider: "mock1"}}
	client.registry.Register("mock1", mock)

	_, err := client.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastRequest.Num != 0 {
		t.Errorf("expected num 0 with nil defaults, got %d", mock.lastRequest.Num)
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

func TestSearchUsesFirstSupportingProvider(t *testing.T) {
	client := New(&Config{})
	noImages := &mockAdapterLimited{
		name:           "no-images",
		supportedTypes: map[SearchType]bool{SearchTypeWeb: true},
	}
	hasImages := &mockAdapter{
		name:    "has-images",
		results: &SearchResponse{Provider: "has-images"},
	}
	client.registry.Register("no-images", noImages)
	client.registry.Register("has-images", hasImages)

	resp, err := client.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeImages,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "has-images" {
		t.Errorf("expected 'has-images', got '%s'", resp.Provider)
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

func TestSearchWithFallbackSpecificProviders(t *testing.T) {
	client := New(&Config{})
	a := &mockAdapter{name: "a", results: &SearchResponse{Provider: "a"}}
	b := &mockAdapter{name: "b", results: &SearchResponse{Provider: "b"}}
	client.registry.Register("a", a)
	client.registry.Register("b", b)

	resp, err := client.SearchWithFallback(context.Background(), SearchRequest{Query: "test"}, []string{"b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "b" {
		t.Errorf("expected provider b, got %s", resp.Provider)
	}
}

func TestSearchWithFallbackNoProviders(t *testing.T) {
	client := New(&Config{})
	_, err := client.SearchWithFallback(context.Background(), SearchRequest{Query: "test"}, nil)
	if err == nil {
		t.Fatal("expected error with empty fallback list")
	}
}

func TestSearchWithFallbackAppliesDefaults(t *testing.T) {
	client := New(&Config{
		Defaults: &SearchDefaults{Num: 25},
	})
	mock := &mockAdapter{name: "m", results: &SearchResponse{Provider: "m"}}
	client.registry.Register("m", mock)

	_, err := client.SearchWithFallback(context.Background(), SearchRequest{Query: "test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastRequest.Num != 25 {
		t.Errorf("expected num 25 from defaults, got %d", mock.lastRequest.Num)
	}
}

func TestGetRegistry(t *testing.T) {
	client := New(&Config{
		Serper: &ProviderConfig{APIKey: "k"},
	})
	reg := client.GetRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	adapter := reg.Get("serper")
	if adapter == nil {
		t.Fatal("expected serper adapter in registry")
	}
	if adapter.Name() != "serper" {
		t.Errorf("expected name 'serper', got '%s'", adapter.Name())
	}
}

func TestVersion(t *testing.T) {
	if Version != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %s", Version)
	}
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mock := &mockAdapter{name: "test"}
	reg.Register("test", mock)

	got := reg.Get("test")
	if got == nil {
		t.Fatal("expected adapter")
	}
	if got.Name() != "test" {
		t.Errorf("expected 'test', got '%s'", got.Name())
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for missing adapter")
	}
}

func TestRegistryOrder(t *testing.T) {
	reg := NewRegistry()
	reg.Register("b", &mockAdapter{name: "b"})
	reg.Register("a", &mockAdapter{name: "a"})
	reg.Register("c", &mockAdapter{name: "c"})

	names := reg.Names()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "b" || names[1] != "a" || names[2] != "c" {
		t.Errorf("unexpected order: %v", names)
	}
}

func TestRegistryAll(t *testing.T) {
	reg := NewRegistry()
	reg.Register("x", &mockAdapter{name: "x"})
	reg.Register("y", &mockAdapter{name: "y"})

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 adapters, got %d", len(all))
	}
	if all[0].Name() != "x" || all[1].Name() != "y" {
		t.Errorf("unexpected order: %s, %s", all[0].Name(), all[1].Name())
	}
}

func TestRegistryOverwrite(t *testing.T) {
	reg := NewRegistry()
	reg.Register("x", &mockAdapter{name: "x-old"})
	reg.Register("x", &mockAdapter{name: "x-new"})

	names := reg.Names()
	if len(names) != 1 {
		t.Fatalf("expected 1 name after overwrite, got %d", len(names))
	}
	if reg.Get("x").Name() != "x-new" {
		t.Errorf("expected overwritten adapter")
	}
}

func TestRegistryNamesReturnsDefensiveCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Register("a", &mockAdapter{name: "a"})

	names := reg.Names()
	names[0] = "modified"

	original := reg.Names()
	if original[0] != "a" {
		t.Error("Names() should return a copy, not a reference")
	}
}

// ---------------------------------------------------------------------------
// Mock adapters
// ---------------------------------------------------------------------------

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

type mockAdapterLimited struct {
	name           string
	supportedTypes map[SearchType]bool
}

func (m *mockAdapterLimited) Name() string { return m.name }

func (m *mockAdapterLimited) SupportsType(t SearchType) bool {
	return m.supportedTypes[t]
}

func (m *mockAdapterLimited) Search(_ context.Context, _ SearchRequest) (*SearchResponse, error) {
	return &SearchResponse{Provider: m.name}, nil
}
