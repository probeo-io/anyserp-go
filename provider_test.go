package anyserp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSerperResponseMapping(t *testing.T) {
	mockResponse := map[string]interface{}{
		"organic": []interface{}{
			map[string]interface{}{
				"title":   "Go Programming Language",
				"link":    "https://go.dev",
				"snippet": "The Go programming language",
				"domain":  "go.dev",
				"date":    "2024-01-01",
			},
			map[string]interface{}{
				"title":   "Go Tutorial",
				"link":    "https://go.dev/tour",
				"snippet": "A tour of Go",
				"domain":  "go.dev",
			},
		},
		"searchParameters": map[string]interface{}{
			"timeTaken": 0.35,
		},
		"relatedSearches": []interface{}{
			map[string]interface{}{"query": "golang tutorial"},
			map[string]interface{}{"query": "go vs rust"},
		},
		"peopleAlsoAsk": []interface{}{
			map[string]interface{}{
				"question": "Is Go a good language?",
				"snippet":  "Yes, Go is great.",
				"title":    "Go Language Review",
				"link":     "https://example.com/review",
			},
		},
		"knowledgeGraph": map[string]interface{}{
			"title":             "Go",
			"type":              "Programming Language",
			"description":       "Go is a programming language",
			"descriptionSource": "Wikipedia",
			"descriptionLink":   "https://en.wikipedia.org/wiki/Go",
			"imageUrl":          "https://go.dev/logo.png",
			"attributes": map[string]interface{}{
				"Developer": "Google",
			},
		},
		"answerBox": map[string]interface{}{
			"snippet": "Go is a statically typed language",
			"title":   "About Go",
			"link":    "https://go.dev/about",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-KEY") != "test-key" {
			t.Error("missing or wrong API key header")
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/search") {
			t.Errorf("expected /search path, got %s", r.URL.Path)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["q"] != "golang" {
			t.Errorf("expected query 'golang', got '%v'", body["q"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create adapter with custom client that rewrites the URL
	adapter := &SerperAdapter{
		apiKey: "test-key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "golang",
		Num:   10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response structure
	if resp.Provider != "serper" {
		t.Errorf("expected provider 'serper', got '%s'", resp.Provider)
	}
	if resp.Query != "golang" {
		t.Errorf("expected query 'golang', got '%s'", resp.Query)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}

	// Verify first result
	r := resp.Results[0]
	if r.Position != 1 {
		t.Errorf("expected position 1, got %d", r.Position)
	}
	if r.Title != "Go Programming Language" {
		t.Errorf("expected title 'Go Programming Language', got '%s'", r.Title)
	}
	if r.URL != "https://go.dev" {
		t.Errorf("expected URL 'https://go.dev', got '%s'", r.URL)
	}
	if r.Domain != "go.dev" {
		t.Errorf("expected domain 'go.dev', got '%s'", r.Domain)
	}
	if r.DatePublished != "2024-01-01" {
		t.Errorf("expected date '2024-01-01', got '%s'", r.DatePublished)
	}

	// Verify search time
	if resp.SearchTime != 350 {
		t.Errorf("expected search time 350ms, got %f", resp.SearchTime)
	}

	// Verify related searches
	if len(resp.RelatedSearches) != 2 {
		t.Fatalf("expected 2 related searches, got %d", len(resp.RelatedSearches))
	}
	if resp.RelatedSearches[0] != "golang tutorial" {
		t.Errorf("expected 'golang tutorial', got '%s'", resp.RelatedSearches[0])
	}

	// Verify PAA
	if len(resp.PeopleAlsoAsk) != 1 {
		t.Fatalf("expected 1 PAA, got %d", len(resp.PeopleAlsoAsk))
	}
	if resp.PeopleAlsoAsk[0].Question != "Is Go a good language?" {
		t.Errorf("unexpected PAA question: %s", resp.PeopleAlsoAsk[0].Question)
	}

	// Verify knowledge panel
	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	if resp.KnowledgePanel.Title != "Go" {
		t.Errorf("expected KP title 'Go', got '%s'", resp.KnowledgePanel.Title)
	}
	if resp.KnowledgePanel.Source != "Wikipedia" {
		t.Errorf("expected KP source 'Wikipedia', got '%s'", resp.KnowledgePanel.Source)
	}
	if resp.KnowledgePanel.Attributes["Developer"] != "Google" {
		t.Errorf("expected KP attribute Developer=Google, got '%s'", resp.KnowledgePanel.Attributes["Developer"])
	}

	// Verify answer box
	if resp.AnswerBox == nil {
		t.Fatal("expected answer box")
	}
	if resp.AnswerBox.Snippet != "Go is a statically typed language" {
		t.Errorf("unexpected answer box snippet: %s", resp.AnswerBox.Snippet)
	}
}

func TestSearchAPIResponseMapping(t *testing.T) {
	mockResponse := map[string]interface{}{
		"organic_results": []interface{}{
			map[string]interface{}{
				"title":   "SearchAPI Result",
				"link":    "https://example.com",
				"snippet": "A search result",
			},
		},
		"search_information": map[string]interface{}{
			"total_results":        float64(1000),
			"time_taken_displayed": "0.45",
		},
		"related_searches": []interface{}{
			map[string]interface{}{"query": "related query"},
		},
		"people_also_ask": []interface{}{
			map[string]interface{}{
				"question": "What is SearchAPI?",
				"snippet":  "SearchAPI is a service",
				"title":    "SearchAPI Info",
				"link":     "https://searchapi.io",
			},
		},
		"knowledge_graph": map[string]interface{}{
			"title":       "SearchAPI",
			"description": "A search API service",
			"source":      map[string]interface{}{"name": "Wikipedia", "link": "https://wikipedia.org"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing or wrong Authorization header")
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	adapter := &SearchAPIAdapter{
		apiKey: "test-key",
		client: newTestClient(server.URL, "https://www.searchapi.io"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Provider != "searchapi" {
		t.Errorf("expected provider 'searchapi', got '%s'", resp.Provider)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Title != "SearchAPI Result" {
		t.Errorf("unexpected title: %s", resp.Results[0].Title)
	}
	if resp.TotalResults != 1000 {
		t.Errorf("expected total results 1000, got %d", resp.TotalResults)
	}
	if resp.SearchTime != 450 {
		t.Errorf("expected search time 450ms, got %f", resp.SearchTime)
	}
	if len(resp.PeopleAlsoAsk) != 1 {
		t.Fatalf("expected 1 PAA, got %d", len(resp.PeopleAlsoAsk))
	}
	if resp.PeopleAlsoAsk[0].Question != "What is SearchAPI?" {
		t.Errorf("unexpected PAA question: %s", resp.PeopleAlsoAsk[0].Question)
	}
	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	if resp.KnowledgePanel.Source != "Wikipedia" {
		t.Errorf("expected KP source 'Wikipedia', got '%s'", resp.KnowledgePanel.Source)
	}
}

func TestSearchAPIWithAiOverview(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		engine := r.URL.Query().Get("engine")
		if engine == "google_ai_overview" {
			// AI overview response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"markdown": "# AI Overview\nThis is an AI overview.",
				"text_blocks": []interface{}{
					map[string]interface{}{
						"type":              "paragraph",
						"answer":            "This is the answer.",
						"reference_indexes": []interface{}{float64(1), float64(2)},
					},
				},
				"reference_links": []interface{}{
					map[string]interface{}{
						"index":  float64(1),
						"title":  "Source 1",
						"link":   "https://source1.com",
						"source": "source1.com",
					},
				},
			})
		} else {
			// Regular search response with AI overview page token
			json.NewEncoder(w).Encode(map[string]interface{}{
				"organic_results": []interface{}{
					map[string]interface{}{
						"title":   "Result",
						"link":    "https://example.com",
						"snippet": "A result",
					},
				},
				"ai_overview": map[string]interface{}{
					"page_token": "test_page_token_123",
				},
			})
		}
	}))
	defer server.Close()

	adapter := &SearchAPIAdapter{
		apiKey: "test-key",
		client: newTestClient(server.URL, "https://www.searchapi.io"),
	}

	// Test with includeAiOverview = true
	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query:             "test",
		IncludeAiOverview: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls (search + AI overview), got %d", callCount)
	}
	if resp.AiOverview == nil {
		t.Fatal("expected AI overview")
	}
	if resp.AiOverview.Markdown != "# AI Overview\nThis is an AI overview." {
		t.Errorf("unexpected markdown: %s", resp.AiOverview.Markdown)
	}
	if resp.AiOverview.PageToken != "test_page_token_123" {
		t.Errorf("unexpected page token: %s", resp.AiOverview.PageToken)
	}
	if len(resp.AiOverview.TextBlocks) != 1 {
		t.Fatalf("expected 1 text block, got %d", len(resp.AiOverview.TextBlocks))
	}
	if resp.AiOverview.TextBlocks[0].Answer != "This is the answer." {
		t.Errorf("unexpected answer: %s", resp.AiOverview.TextBlocks[0].Answer)
	}
	if len(resp.AiOverview.TextBlocks[0].ReferenceIndexes) != 2 {
		t.Errorf("expected 2 reference indexes, got %d", len(resp.AiOverview.TextBlocks[0].ReferenceIndexes))
	}
	if len(resp.AiOverview.References) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(resp.AiOverview.References))
	}
	if resp.AiOverview.References[0].Title != "Source 1" {
		t.Errorf("unexpected reference title: %s", resp.AiOverview.References[0].Title)
	}
}

func TestSearchAPIWithPageTokenOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"organic_results": []interface{}{},
			"ai_overview": map[string]interface{}{
				"page_token": "token_abc",
			},
		})
	}))
	defer server.Close()

	adapter := &SearchAPIAdapter{
		apiKey: "test-key",
		client: newTestClient(server.URL, "https://www.searchapi.io"),
	}

	// Test with includeAiOverview = false (default)
	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.AiOverview == nil {
		t.Fatal("expected AI overview with page token")
	}
	if resp.AiOverview.PageToken != "token_abc" {
		t.Errorf("expected page token 'token_abc', got '%s'", resp.AiOverview.PageToken)
	}
	if len(resp.AiOverview.TextBlocks) != 0 {
		t.Errorf("expected empty text blocks, got %d", len(resp.AiOverview.TextBlocks))
	}
}

func TestSerperErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Invalid API key",
		})
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "bad-key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	asErr, ok := err.(*AnySerpError)
	if !ok {
		t.Fatalf("expected AnySerpError, got %T", err)
	}
	if asErr.Code != 401 {
		t.Errorf("expected code 401, got %d", asErr.Code)
	}
	if asErr.Message != "Invalid API key" {
		t.Errorf("unexpected error message: %s", asErr.Message)
	}
}

func TestSupportsType(t *testing.T) {
	tests := []struct {
		name     string
		adapter  SearchAdapter
		typ      SearchType
		expected bool
	}{
		{"serper web", NewSerperAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"serper images", NewSerperAdapter("key", http.DefaultClient), SearchTypeImages, true},
		{"google web", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeWeb, true},
		{"google images", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeImages, true},
		{"google news", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeNews, false},
		{"google videos", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeVideos, false},
		{"dataforseo web", NewDataForSeoAdapter("l", "p", http.DefaultClient), SearchTypeWeb, true},
		{"dataforseo news", NewDataForSeoAdapter("l", "p", http.DefaultClient), SearchTypeNews, true},
		{"dataforseo images", NewDataForSeoAdapter("l", "p", http.DefaultClient), SearchTypeImages, false},
		{"scrapingdog web", NewScrapingDogAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"scrapingdog videos", NewScrapingDogAdapter("key", http.DefaultClient), SearchTypeVideos, false},
		{"searchcans web", NewSearchCansAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"searchcans news", NewSearchCansAdapter("key", http.DefaultClient), SearchTypeNews, true},
		{"searchcans images", NewSearchCansAdapter("key", http.DefaultClient), SearchTypeImages, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.adapter.SupportsType(tc.typ)
			if got != tc.expected {
				t.Errorf("SupportsType(%s) = %v, want %v", tc.typ, got, tc.expected)
			}
		})
	}
}

// newTestClient creates an HTTP client that rewrites requests from the real
// base URL to the test server URL.
func newTestClient(testServerURL, realBaseURL string) *http.Client {
	return &http.Client{
		Transport: &urlRewriteTransport{
			testURL: testServerURL,
			realURL: realBaseURL,
			base:    http.DefaultTransport,
		},
	}
}

type urlRewriteTransport struct {
	testURL string
	realURL string
	base    http.RoundTripper
}

func (t *urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	original := req.URL.String()
	if strings.HasPrefix(original, t.realURL) {
		newURL := t.testURL + original[len(t.realURL):]
		parsedURL, err := http.NewRequest(req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		parsedURL.Header = req.Header
		return t.base.RoundTrip(parsedURL)
	}
	return t.base.RoundTrip(req)
}
