package anyserp

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

// Registry manages search provider adapters.
type Registry struct {
	adapters map[string]SearchAdapter
	order    []string
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]SearchAdapter),
	}
}

// Register adds an adapter to the registry.
func (r *Registry) Register(name string, adapter SearchAdapter) {
	if _, exists := r.adapters[name]; !exists {
		r.order = append(r.order, name)
	}
	r.adapters[name] = adapter
}

// Get returns an adapter by name.
func (r *Registry) Get(name string) SearchAdapter {
	return r.adapters[name]
}

// All returns all registered adapters in registration order.
func (r *Registry) All() []SearchAdapter {
	out := make([]SearchAdapter, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.adapters[name])
	}
	return out
}

// Names returns the names of all registered adapters in registration order.
func (r *Registry) Names() []string {
	return append([]string{}, r.order...)
}

// AnySerp is the main client for searching across multiple providers.
type AnySerp struct {
	registry *Registry
	defaults *SearchDefaults
	aliases  map[string]string
	client   *http.Client
}

// New creates a new AnySerp client. If config is nil, providers are loaded from
// environment variables only.
func New(config *Config) *AnySerp {
	if config == nil {
		config = &Config{}
	}

	client := &http.Client{Timeout: 30 * time.Second}

	a := &AnySerp{
		registry: NewRegistry(),
		defaults: config.Defaults,
		aliases:  config.Aliases,
		client:   client,
	}

	if a.aliases == nil {
		a.aliases = make(map[string]string)
	}

	a.registerProviders(config)
	return a
}

func (a *AnySerp) registerProviders(config *Config) {
	// Serper
	if key := configOrEnv(config.Serper, "SERPER_API_KEY"); key != "" {
		a.registry.Register("serper", NewSerperAdapter(key, a.client))
	}

	// SerpAPI
	if key := configOrEnv(config.SerpAPI, "SERPAPI_API_KEY"); key != "" {
		a.registry.Register("serpapi", NewSerpAPIAdapter(key, a.client))
	}

	// Google CSE
	googleKey := configOrEnv(config.Google, "GOOGLE_CSE_API_KEY")
	engineID := ""
	if config.Google != nil {
		engineID = config.Google.EngineID
	}
	if engineID == "" {
		engineID = os.Getenv("GOOGLE_CSE_ENGINE_ID")
	}
	if googleKey != "" && engineID != "" {
		a.registry.Register("google", NewGoogleAdapter(googleKey, engineID, a.client))
	}

	// Bing
	if key := configOrEnv(config.Bing, "BING_API_KEY"); key != "" {
		a.registry.Register("bing", NewBingAdapter(key, a.client))
	}

	// Brave
	if key := configOrEnv(config.Brave, "BRAVE_API_KEY"); key != "" {
		a.registry.Register("brave", NewBraveAdapter(key, a.client))
	}

	// DataForSEO
	dfLogin := ""
	dfPassword := ""
	if config.DataForSEO != nil {
		dfLogin = config.DataForSEO.Login
		dfPassword = config.DataForSEO.Password
	}
	if dfLogin == "" {
		dfLogin = os.Getenv("DATAFORSEO_LOGIN")
	}
	if dfPassword == "" {
		dfPassword = os.Getenv("DATAFORSEO_PASSWORD")
	}
	if dfLogin != "" && dfPassword != "" {
		a.registry.Register("dataforseo", NewDataForSeoAdapter(dfLogin, dfPassword, a.client))
	}

	// SearchAPI
	if key := configOrEnv(config.SearchAPI, "SEARCHAPI_API_KEY"); key != "" {
		a.registry.Register("searchapi", NewSearchAPIAdapter(key, a.client))
	}

	// ValueSERP
	if key := configOrEnv(config.ValueSERP, "VALUESERP_API_KEY"); key != "" {
		a.registry.Register("valueserp", NewValueSerpAdapter(key, a.client))
	}

	// ScrapingDog
	if key := configOrEnv(config.ScrapingDog, "SCRAPINGDOG_API_KEY"); key != "" {
		a.registry.Register("scrapingdog", NewScrapingDogAdapter(key, a.client))
	}

	// BrightData
	if key := configOrEnv(config.BrightData, "BRIGHTDATA_API_KEY"); key != "" {
		a.registry.Register("brightdata", NewBrightDataAdapter(key, a.client))
	}

	// SearchCans
	if key := configOrEnv(config.SearchCans, "SEARCHCANS_API_KEY"); key != "" {
		a.registry.Register("searchcans", NewSearchCansAdapter(key, a.client))
	}
}

func configOrEnv(pc *ProviderConfig, envVar string) string {
	if pc != nil && pc.APIKey != "" {
		return pc.APIKey
	}
	return os.Getenv(envVar)
}

// Search executes a search using the first available provider that supports the
// requested search type. If the query contains a provider prefix (e.g. "serper/golang"),
// that provider is used directly.
func (a *AnySerp) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	a.applyDefaults(&request)

	// Check for provider prefix in query
	var providerName string
	if idx := strings.Index(request.Query, "/"); idx > 0 {
		maybeProvider := request.Query[:idx]
		resolved := a.resolveAlias(maybeProvider)
		if a.registry.Get(resolved) != nil {
			providerName = resolved
			request.Query = request.Query[idx+1:]
		}
	}

	if providerName != "" {
		adapter := a.registry.Get(providerName)
		if adapter == nil {
			return nil, NewAnySerpError(400, "Provider \""+providerName+"\" not configured",
				map[string]interface{}{"provider_name": providerName})
		}
		return adapter.Search(ctx, request)
	}

	// No provider specified - use first available that supports the type
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}
	for _, adapter := range a.registry.All() {
		if adapter.SupportsType(searchType) {
			return adapter.Search(ctx, request)
		}
	}

	return nil, NewAnySerpError(400, "No provider configured. Set an API key for at least one provider.", nil)
}

// SearchWithFallback tries providers in order, falling back on error.
func (a *AnySerp) SearchWithFallback(ctx context.Context, request SearchRequest, providers []string) (*SearchResponse, error) {
	a.applyDefaults(&request)

	if len(providers) == 0 {
		providers = a.registry.Names()
	}

	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}

	var lastErr error
	for _, name := range providers {
		adapter := a.registry.Get(name)
		if adapter == nil || !adapter.SupportsType(searchType) {
			continue
		}
		resp, err := adapter.Search(ctx, request)
		if err != nil {
			lastErr = err
			continue
		}
		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, NewAnySerpError(400, "No providers available for fallback", nil)
}

// Providers returns the names of all configured providers.
func (a *AnySerp) Providers() []string {
	return a.registry.Names()
}

// GetRegistry returns the underlying registry for direct adapter access.
func (a *AnySerp) GetRegistry() *Registry {
	return a.registry
}

func (a *AnySerp) applyDefaults(req *SearchRequest) {
	if a.defaults == nil {
		return
	}
	if req.Num == 0 && a.defaults.Num != 0 {
		req.Num = a.defaults.Num
	}
	if req.Country == "" && a.defaults.Country != "" {
		req.Country = a.defaults.Country
	}
	if req.Language == "" && a.defaults.Language != "" {
		req.Language = a.defaults.Language
	}
	if !req.Safe && a.defaults.Safe {
		req.Safe = a.defaults.Safe
	}
}

func (a *AnySerp) resolveAlias(name string) string {
	if alias, ok := a.aliases[name]; ok {
		return alias
	}
	return name
}
