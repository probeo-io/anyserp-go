package anyserp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Serper
// ---------------------------------------------------------------------------

func TestSerperResponseMapping(t *testing.T) {
	mockResponse := serperMockWebResponse()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHeader(t, r, "X-API-KEY", "test-key")
		assertMethod(t, r, http.MethodPost)
		assertPathSuffix(t, r, "/search")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["q"] != "golang" {
			t.Errorf("expected query 'golang', got '%v'", body["q"])
		}

		writeJSON(w, mockResponse)
	}))
	defer server.Close()

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

	assertProvider(t, resp, "serper")
	assertQuery(t, resp, "golang")
	assertResultCount(t, resp, 2)
	assertFirstResult(t, resp, "Go Programming Language", "https://go.dev")

	if resp.Results[0].Domain != "go.dev" {
		t.Errorf("expected domain 'go.dev', got '%s'", resp.Results[0].Domain)
	}
	if resp.Results[0].DatePublished != "2024-01-01" {
		t.Errorf("expected date '2024-01-01', got '%s'", resp.Results[0].DatePublished)
	}
	if resp.SearchTime != 350 {
		t.Errorf("expected search time 350ms, got %f", resp.SearchTime)
	}
	assertRelatedSearchCount(t, resp, 2)
	assertPAACount(t, resp, 1)

	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	if resp.KnowledgePanel.Title != "Go" {
		t.Errorf("expected KP title 'Go', got '%s'", resp.KnowledgePanel.Title)
	}
	if resp.KnowledgePanel.Attributes["Developer"] != "Google" {
		t.Errorf("expected Developer=Google, got '%s'", resp.KnowledgePanel.Attributes["Developer"])
	}
	if resp.AnswerBox == nil {
		t.Fatal("expected answer box")
	}
	if resp.AnswerBox.Snippet != "Go is a statically typed language" {
		t.Errorf("unexpected answer box snippet: %s", resp.AnswerBox.Snippet)
	}
}

func TestSerperImagesSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPathSuffix(t, r, "/images")
		writeJSON(w, map[string]interface{}{
			"images": []interface{}{
				map[string]interface{}{
					"title":        "Test Image",
					"link":         "https://example.com/page",
					"imageUrl":     "https://example.com/img.jpg",
					"imageWidth":   float64(800),
					"imageHeight":  float64(600),
					"thumbnailUrl": "https://example.com/thumb.jpg",
					"domain":       "example.com",
				},
			},
		})
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeImages,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	r := resp.Results[0]
	if r.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("unexpected imageURL: %s", r.ImageURL)
	}
	if r.ImageWidth != 800 || r.ImageHeight != 600 {
		t.Errorf("unexpected dimensions: %dx%d", r.ImageWidth, r.ImageHeight)
	}
}

func TestSerperNewsSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPathSuffix(t, r, "/news")
		writeJSON(w, map[string]interface{}{
			"news": []interface{}{
				map[string]interface{}{
					"title":   "Breaking News",
					"link":    "https://news.example.com",
					"snippet": "News snippet",
					"source":  "News Source",
					"date":    "2024-06-01",
				},
			},
		})
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeNews,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	if resp.Results[0].Source != "News Source" {
		t.Errorf("expected source 'News Source', got '%s'", resp.Results[0].Source)
	}
}

func TestSerperVideosSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPathSuffix(t, r, "/videos")
		writeJSON(w, map[string]interface{}{
			"videos": []interface{}{
				map[string]interface{}{
					"title":    "Test Video",
					"link":     "https://youtube.com/watch?v=123",
					"snippet":  "Video desc",
					"duration": "10:30",
					"channel":  "TestChannel",
					"date":     "2024-03-01",
				},
			},
		})
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeVideos,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	r := resp.Results[0]
	if r.Duration != "10:30" {
		t.Errorf("expected duration '10:30', got '%s'", r.Duration)
	}
	if r.Channel != "TestChannel" {
		t.Errorf("expected channel 'TestChannel', got '%s'", r.Channel)
	}
}

func TestSerperQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["gl"] != "us" {
			t.Errorf("expected gl=us, got %v", body["gl"])
		}
		if body["hl"] != "en" {
			t.Errorf("expected hl=en, got %v", body["hl"])
		}
		if body["num"] != float64(20) {
			t.Errorf("expected num=20, got %v", body["num"])
		}
		if body["page"] != float64(2) {
			t.Errorf("expected page=2, got %v", body["page"])
		}
		if body["tbs"] != "qdr:w" {
			t.Errorf("expected tbs=qdr:w, got %v", body["tbs"])
		}

		writeJSON(w, map[string]interface{}{"organic": []interface{}{}})
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	adapter.Search(context.Background(), SearchRequest{
		Query:     "test",
		Num:       20,
		Page:      2,
		Country:   "us",
		Language:  "en",
		DateRange: DateRangeWeek,
	})
}

func TestSerperErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]interface{}{"message": "Invalid API key"})
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "bad-key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 401)
}

// ---------------------------------------------------------------------------
// SerpAPI
// ---------------------------------------------------------------------------

func TestSerpAPIResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)

		q := r.URL.Query()
		if q.Get("api_key") != "test-key" {
			t.Error("missing api_key param")
		}
		if q.Get("engine") != "google" {
			t.Errorf("expected engine=google, got %s", q.Get("engine"))
		}

		writeJSON(w, map[string]interface{}{
			"organic_results": []interface{}{
				map[string]interface{}{
					"title":          "SerpAPI Result",
					"link":           "https://example.com",
					"snippet":        "A result",
					"displayed_link": "example.com",
					"position":       float64(1),
				},
			},
			"search_information": map[string]interface{}{
				"total_results":        float64(5000),
				"time_taken_displayed": "0.5",
			},
			"related_searches": []interface{}{
				map[string]interface{}{"query": "related"},
			},
			"related_questions": []interface{}{
				map[string]interface{}{
					"question": "What is SerpAPI?",
					"snippet":  "SerpAPI is a service",
				},
			},
			"knowledge_graph": map[string]interface{}{
				"title":       "SerpAPI",
				"description": "A search API",
				"source": map[string]interface{}{
					"name": "Wikipedia",
					"link": "https://wikipedia.org",
				},
			},
			"answer_box": map[string]interface{}{
				"answer": "SerpAPI answer",
				"title":  "About SerpAPI",
				"link":   "https://serpapi.com",
			},
		})
	}))
	defer server.Close()

	adapter := &SerpAPIAdapter{
		apiKey: "test-key",
		client: newTestClient(server.URL, "https://serpapi.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "serpapi")
	assertResultCount(t, resp, 1)
	if resp.TotalResults != 5000 {
		t.Errorf("expected 5000 total, got %d", resp.TotalResults)
	}
	if resp.SearchTime != 500 {
		t.Errorf("expected 500ms, got %f", resp.SearchTime)
	}
	assertRelatedSearchCount(t, resp, 1)
	assertPAACount(t, resp, 1)

	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	if resp.KnowledgePanel.Source != "Wikipedia" {
		t.Errorf("expected source Wikipedia, got %s", resp.KnowledgePanel.Source)
	}
	if resp.AnswerBox == nil {
		t.Fatal("expected answer box")
	}
	if resp.AnswerBox.Snippet != "SerpAPI answer" {
		t.Errorf("unexpected answer box: %s", resp.AnswerBox.Snippet)
	}
}

func TestSerpAPIQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("gl") != "gb" {
			t.Errorf("expected gl=gb, got %s", q.Get("gl"))
		}
		if q.Get("hl") != "en" {
			t.Errorf("expected hl=en, got %s", q.Get("hl"))
		}
		if q.Get("safe") != "active" {
			t.Errorf("expected safe=active, got %s", q.Get("safe"))
		}
		if q.Get("num") != "5" {
			t.Errorf("expected num=5, got %s", q.Get("num"))
		}
		if q.Get("tbs") != "qdr:m" {
			t.Errorf("expected tbs=qdr:m, got %s", q.Get("tbs"))
		}
		writeJSON(w, map[string]interface{}{"organic_results": []interface{}{}})
	}))
	defer server.Close()

	adapter := &SerpAPIAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://serpapi.com"),
	}

	adapter.Search(context.Background(), SearchRequest{
		Query:     "test",
		Num:       5,
		Country:   "gb",
		Language:  "en",
		Safe:      true,
		DateRange: DateRangeMonth,
	})
}

func TestSerpAPIPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("start") != "20" {
			t.Errorf("expected start=20 (page 3, num 10), got %s", q.Get("start"))
		}
		writeJSON(w, map[string]interface{}{"organic_results": []interface{}{}})
	}))
	defer server.Close()

	adapter := &SerpAPIAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://serpapi.com"),
	}

	adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Page:  3,
	})
}

func TestSerpAPIErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, map[string]interface{}{"error": "Invalid API key"})
	}))
	defer server.Close()

	adapter := &SerpAPIAdapter{
		apiKey: "bad",
		client: newTestClient(server.URL, "https://serpapi.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 403)
}

// ---------------------------------------------------------------------------
// Google CSE
// ---------------------------------------------------------------------------

func TestGoogleResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("key") != "gkey" {
			t.Errorf("expected key=gkey, got %s", q.Get("key"))
		}
		if q.Get("cx") != "engine1" {
			t.Errorf("expected cx=engine1, got %s", q.Get("cx"))
		}

		writeJSON(w, map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"title":       "Google Result",
					"link":        "https://example.com",
					"snippet":     "A snippet",
					"displayLink": "example.com",
				},
			},
			"searchInformation": map[string]interface{}{
				"totalResults": "10000",
				"searchTime":   float64(0.25),
			},
		})
	}))
	defer server.Close()

	adapter := &GoogleAdapter{
		apiKey:   "gkey",
		engineID: "engine1",
		client:   newTestClient(server.URL, "https://www.googleapis.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "google")
	assertResultCount(t, resp, 1)
	if resp.TotalResults != 10000 {
		t.Errorf("expected 10000, got %d", resp.TotalResults)
	}
	if resp.SearchTime != 250 {
		t.Errorf("expected 250ms, got %f", resp.SearchTime)
	}
}

func TestGoogleImageSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("searchType") != "image" {
			t.Errorf("expected searchType=image, got %s", q.Get("searchType"))
		}
		writeJSON(w, map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"title": "Image",
					"link":  "https://example.com/img.jpg",
					"image": map[string]interface{}{
						"width":         float64(1920),
						"height":        float64(1080),
						"thumbnailLink": "https://example.com/thumb.jpg",
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &GoogleAdapter{
		apiKey:   "key",
		engineID: "cx",
		client:   newTestClient(server.URL, "https://www.googleapis.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeImages,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	r := resp.Results[0]
	if r.ImageWidth != 1920 || r.ImageHeight != 1080 {
		t.Errorf("unexpected dimensions: %dx%d", r.ImageWidth, r.ImageHeight)
	}
}

func TestGoogleNumCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("num") != "10" {
			t.Errorf("expected num capped at 10, got %s", q.Get("num"))
		}
		writeJSON(w, map[string]interface{}{"items": []interface{}{}})
	}))
	defer server.Close()

	adapter := &GoogleAdapter{
		apiKey:   "key",
		engineID: "cx",
		client:   newTestClient(server.URL, "https://www.googleapis.com"),
	}

	adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Num:   50,
	})
}

func TestGoogleErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Daily limit exceeded",
			},
		})
	}))
	defer server.Close()

	adapter := &GoogleAdapter{
		apiKey:   "key",
		engineID: "cx",
		client:   newTestClient(server.URL, "https://www.googleapis.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	asErr := assertAnySerpError(t, err, 403)
	if asErr != nil && asErr.Message != "Daily limit exceeded" {
		t.Errorf("expected 'Daily limit exceeded', got %q", asErr.Message)
	}
}

// ---------------------------------------------------------------------------
// Bing
// ---------------------------------------------------------------------------

func TestBingResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertHeader(t, r, "Ocp-Apim-Subscription-Key", "bing-key")
		assertPathSuffix(t, r, "/search")

		writeJSON(w, map[string]interface{}{
			"webPages": map[string]interface{}{
				"totalEstimatedMatches": float64(50000),
				"value": []interface{}{
					map[string]interface{}{
						"name":            "Bing Result",
						"url":             "https://example.com",
						"snippet":         "Bing snippet",
						"dateLastCrawled": "2024-05-15",
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &BingAdapter{
		apiKey: "bing-key",
		client: newTestClient(server.URL, "https://api.bing.microsoft.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "bing")
	assertResultCount(t, resp, 1)
	if resp.TotalResults != 50000 {
		t.Errorf("expected 50000, got %d", resp.TotalResults)
	}
	if resp.Results[0].DatePublished != "2024-05-15" {
		t.Errorf("unexpected date: %s", resp.Results[0].DatePublished)
	}
}

func TestBingImageSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPathSuffix(t, r, "/images/search")
		writeJSON(w, map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{
					"name":         "Bing Image",
					"hostPageUrl":  "https://example.com/page",
					"contentUrl":   "https://example.com/img.jpg",
					"width":        float64(640),
					"height":       float64(480),
					"thumbnailUrl": "https://example.com/thumb.jpg",
				},
			},
		})
	}))
	defer server.Close()

	adapter := &BingAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://api.bing.microsoft.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeImages,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	r := resp.Results[0]
	if r.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("unexpected imageURL: %s", r.ImageURL)
	}
	if r.Domain != "example.com" {
		t.Errorf("expected domain 'example.com', got '%s'", r.Domain)
	}
}

func TestBingNewsSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPathSuffix(t, r, "/news/search")
		writeJSON(w, map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{
					"name":          "Bing News",
					"url":           "https://news.example.com",
					"description":   "News desc",
					"datePublished": "2024-06-01",
					"provider": []interface{}{
						map[string]interface{}{"name": "CNN"},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &BingAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://api.bing.microsoft.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeNews,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	if resp.Results[0].Source != "CNN" {
		t.Errorf("expected source 'CNN', got '%s'", resp.Results[0].Source)
	}
}

func TestBingVideoSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPathSuffix(t, r, "/videos/search")
		writeJSON(w, map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{
					"name":         "Bing Video",
					"contentUrl":   "https://youtube.com/watch?v=1",
					"description":  "Video desc",
					"duration":     "PT5M30S",
					"thumbnailUrl": "https://example.com/thumb.jpg",
					"creator":      map[string]interface{}{"name": "Channel1"},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &BingAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://api.bing.microsoft.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeVideos,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	if resp.Results[0].Channel != "Channel1" {
		t.Errorf("expected channel 'Channel1', got '%s'", resp.Results[0].Channel)
	}
}

func TestBingQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("cc") != "us" {
			t.Errorf("expected cc=us, got %s", q.Get("cc"))
		}
		if q.Get("setLang") != "en" {
			t.Errorf("expected setLang=en, got %s", q.Get("setLang"))
		}
		if q.Get("safeSearch") != "Strict" {
			t.Errorf("expected safeSearch=Strict, got %s", q.Get("safeSearch"))
		}
		if q.Get("freshness") != "Week" {
			t.Errorf("expected freshness=Week, got %s", q.Get("freshness"))
		}
		if q.Get("count") != "15" {
			t.Errorf("expected count=15, got %s", q.Get("count"))
		}
		writeJSON(w, map[string]interface{}{"webPages": map[string]interface{}{"value": []interface{}{}}})
	}))
	defer server.Close()

	adapter := &BingAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://api.bing.microsoft.com"),
	}

	adapter.Search(context.Background(), SearchRequest{
		Query:     "test",
		Num:       15,
		Country:   "us",
		Language:  "en",
		Safe:      true,
		DateRange: DateRangeWeek,
	})
}

func TestBingErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]interface{}{
			"error": map[string]interface{}{"message": "Invalid key"},
		})
	}))
	defer server.Close()

	adapter := &BingAdapter{
		apiKey: "bad",
		client: newTestClient(server.URL, "https://api.bing.microsoft.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 401)
}

// ---------------------------------------------------------------------------
// Brave
// ---------------------------------------------------------------------------

func TestBraveResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		assertHeader(t, r, "X-Subscription-Token", "brave-key")

		writeJSON(w, map[string]interface{}{
			"web": map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"title":       "Brave Result",
						"url":         "https://example.com",
						"description": "Brave desc",
						"meta_url":    map[string]interface{}{"hostname": "example.com"},
						"page_age":    "2024-05-01",
					},
				},
			},
			"query": map[string]interface{}{
				"related_searches": []interface{}{
					map[string]interface{}{"query": "related brave"},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &BraveAdapter{
		apiKey: "brave-key",
		client: newTestClient(server.URL, "https://api.search.brave.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "brave")
	assertResultCount(t, resp, 1)
	assertRelatedSearchCount(t, resp, 1)
	if resp.Results[0].Domain != "example.com" {
		t.Errorf("expected domain 'example.com', got '%s'", resp.Results[0].Domain)
	}
}

func TestBraveQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("country") != "us" {
			t.Errorf("expected country=us, got %s", q.Get("country"))
		}
		if q.Get("search_lang") != "en" {
			t.Errorf("expected search_lang=en, got %s", q.Get("search_lang"))
		}
		if q.Get("safesearch") != "strict" {
			t.Errorf("expected safesearch=strict, got %s", q.Get("safesearch"))
		}
		if q.Get("freshness") != "pd" {
			t.Errorf("expected freshness=pd, got %s", q.Get("freshness"))
		}
		writeJSON(w, map[string]interface{}{"web": map[string]interface{}{"results": []interface{}{}}})
	}))
	defer server.Close()

	adapter := &BraveAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://api.search.brave.com"),
	}

	adapter.Search(context.Background(), SearchRequest{
		Query:     "test",
		Country:   "us",
		Language:  "en",
		Safe:      true,
		DateRange: DateRangeDay,
	})
}

func TestBraveErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		writeJSON(w, map[string]interface{}{"message": "Rate limited"})
	}))
	defer server.Close()

	adapter := &BraveAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://api.search.brave.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 429)
}

// ---------------------------------------------------------------------------
// DataForSEO
// ---------------------------------------------------------------------------

func TestDataForSeoResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			t.Error("missing Basic auth header")
		}

		writeJSON(w, map[string]interface{}{
			"status_code": float64(20000),
			"tasks": []interface{}{
				map[string]interface{}{
					"status_code": float64(20000),
					"result": []interface{}{
						map[string]interface{}{
							"se_results_count": float64(100000),
							"items": []interface{}{
								map[string]interface{}{
									"type":        "organic",
									"title":       "DFS Result",
									"url":         "https://example.com",
									"description": "DFS desc",
									"domain":      "example.com",
								},
								map[string]interface{}{
									"type":        "knowledge_graph",
									"title":       "KG Title",
									"sub_title":   "KG Type",
									"description": "KG desc",
								},
								map[string]interface{}{
									"type":        "featured_snippet",
									"description": "Featured answer",
									"title":       "Featured Title",
									"url":         "https://example.com/featured",
								},
								map[string]interface{}{
									"type": "people_also_ask",
									"items": []interface{}{
										map[string]interface{}{
											"title":       "PAA Question?",
											"description": "PAA answer",
										},
									},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	adapter := &DataForSeoAdapter{
		authHeader: "Basic dGVzdDp0ZXN0",
		client:     newTestClient(server.URL, "https://api.dataforseo.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "dataforseo")
	assertResultCount(t, resp, 1)
	if resp.TotalResults != 100000 {
		t.Errorf("expected 100000, got %d", resp.TotalResults)
	}
	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	if resp.KnowledgePanel.Title != "KG Title" {
		t.Errorf("unexpected KP title: %s", resp.KnowledgePanel.Title)
	}
	if resp.AnswerBox == nil {
		t.Fatal("expected answer box")
	}
	if resp.AnswerBox.Snippet != "Featured answer" {
		t.Errorf("unexpected answer box: %s", resp.AnswerBox.Snippet)
	}
	assertPAACount(t, resp, 1)
}

func TestDataForSeoTaskError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"status_code": float64(20000),
			"tasks": []interface{}{
				map[string]interface{}{
					"status_code":    float64(40501),
					"status_message": "Task quota exceeded",
				},
			},
		})
	}))
	defer server.Close()

	adapter := &DataForSeoAdapter{
		authHeader: "Basic dGVzdDp0ZXN0",
		client:     newTestClient(server.URL, "https://api.dataforseo.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 400)
}

func TestDataForSeoTopLevelError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"status_code":    float64(50000),
			"status_message": "Internal error",
			"tasks":          []interface{}{},
		})
	}))
	defer server.Close()

	adapter := &DataForSeoAdapter{
		authHeader: "Basic dGVzdDp0ZXN0",
		client:     newTestClient(server.URL, "https://api.dataforseo.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 502)
}

// ---------------------------------------------------------------------------
// SearchAPI
// ---------------------------------------------------------------------------

func TestSearchAPIResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Error("missing Bearer auth header")
		}
		assertMethod(t, r, http.MethodGet)

		writeJSON(w, map[string]interface{}{
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
				},
			},
			"knowledge_graph": map[string]interface{}{
				"title":       "SearchAPI",
				"description": "A search API service",
				"source":      map[string]interface{}{"name": "Wikipedia", "link": "https://wikipedia.org"},
			},
		})
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

	assertProvider(t, resp, "searchapi")
	assertResultCount(t, resp, 1)
	if resp.TotalResults != 1000 {
		t.Errorf("expected 1000, got %d", resp.TotalResults)
	}
	if resp.SearchTime != 450 {
		t.Errorf("expected 450ms, got %f", resp.SearchTime)
	}
	assertPAACount(t, resp, 1)
	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	if resp.KnowledgePanel.Source != "Wikipedia" {
		t.Errorf("expected source Wikipedia, got %s", resp.KnowledgePanel.Source)
	}
}

func TestSearchAPIWithAiOverview(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		engine := r.URL.Query().Get("engine")
		if engine == "google_ai_overview" {
			writeJSON(w, map[string]interface{}{
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
			writeJSON(w, map[string]interface{}{
				"organic_results": []interface{}{
					map[string]interface{}{
						"title": "Result", "link": "https://example.com", "snippet": "A result",
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

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query:             "test",
		IncludeAiOverview: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
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
	if len(resp.AiOverview.TextBlocks[0].ReferenceIndexes) != 2 {
		t.Errorf("expected 2 ref indexes, got %d", len(resp.AiOverview.TextBlocks[0].ReferenceIndexes))
	}
	if len(resp.AiOverview.References) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(resp.AiOverview.References))
	}
}

func TestSearchAPIWithPageTokenOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
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

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.AiOverview == nil {
		t.Fatal("expected AI overview with page token")
	}
	if resp.AiOverview.PageToken != "token_abc" {
		t.Errorf("expected 'token_abc', got '%s'", resp.AiOverview.PageToken)
	}
	if len(resp.AiOverview.TextBlocks) != 0 {
		t.Errorf("expected empty text blocks, got %d", len(resp.AiOverview.TextBlocks))
	}
}

func TestSearchAPIErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]interface{}{"error": "Invalid API key"})
	}))
	defer server.Close()

	adapter := &SearchAPIAdapter{
		apiKey: "bad",
		client: newTestClient(server.URL, "https://www.searchapi.io"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 401)
}

// ---------------------------------------------------------------------------
// ValueSERP
// ---------------------------------------------------------------------------

func TestValueSerpResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("api_key") != "vs-key" {
			t.Errorf("expected api_key=vs-key, got %s", q.Get("api_key"))
		}

		writeJSON(w, map[string]interface{}{
			"organic_results": []interface{}{
				map[string]interface{}{
					"title":   "ValueSERP Result",
					"link":    "https://example.com",
					"snippet": "VS desc",
					"domain":  "example.com",
				},
			},
			"search_information": map[string]interface{}{
				"total_results":        float64(8000),
				"time_taken_displayed": "0.3",
			},
			"related_searches": []interface{}{
				map[string]interface{}{"query": "vs related"},
			},
			"people_also_ask": []interface{}{
				map[string]interface{}{"question": "VS PAA?"},
			},
			"knowledge_graph": map[string]interface{}{
				"title":  "VS KG",
				"source": map[string]interface{}{"name": "Wiki"},
			},
			"answer_box": map[string]interface{}{
				"answer": "VS answer",
			},
		})
	}))
	defer server.Close()

	adapter := &ValueSerpAdapter{
		apiKey: "vs-key",
		client: newTestClient(server.URL, "https://api.valueserp.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "valueserp")
	assertResultCount(t, resp, 1)
	if resp.TotalResults != 8000 {
		t.Errorf("expected 8000, got %d", resp.TotalResults)
	}
	if resp.SearchTime != 300 {
		t.Errorf("expected 300ms, got %f", resp.SearchTime)
	}
	assertRelatedSearchCount(t, resp, 1)
	assertPAACount(t, resp, 1)
	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	if resp.AnswerBox == nil {
		t.Fatal("expected answer box")
	}
	if resp.AnswerBox.Snippet != "VS answer" {
		t.Errorf("unexpected answer: %s", resp.AnswerBox.Snippet)
	}
}

func TestValueSerpErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, map[string]interface{}{"error": "Forbidden"})
	}))
	defer server.Close()

	adapter := &ValueSerpAdapter{
		apiKey: "bad",
		client: newTestClient(server.URL, "https://api.valueserp.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 403)
}

// ---------------------------------------------------------------------------
// ScrapingDog
// ---------------------------------------------------------------------------

func TestScrapingDogResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodGet)
		q := r.URL.Query()
		if q.Get("api_key") != "sd-key" {
			t.Errorf("expected api_key=sd-key, got %s", q.Get("api_key"))
		}

		writeJSON(w, map[string]interface{}{
			"organic_results": []interface{}{
				map[string]interface{}{
					"title":          "SD Result",
					"link":           "https://example.com",
					"snippet":        "SD desc",
					"displayed_link": "example.com",
				},
			},
			"people_also_ask": []interface{}{
				map[string]interface{}{"question": "SD PAA?"},
			},
		})
	}))
	defer server.Close()

	adapter := &ScrapingDogAdapter{
		apiKey: "sd-key",
		client: newTestClient(server.URL, "https://api.scrapingdog.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "scrapingdog")
	assertResultCount(t, resp, 1)
	assertPAACount(t, resp, 1)
}

func TestScrapingDogImageSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"image_results": []interface{}{
				map[string]interface{}{
					"title":    "SD Image",
					"link":     "https://example.com/page",
					"original": "https://example.com/img.jpg",
				},
			},
		})
	}))
	defer server.Close()

	adapter := &ScrapingDogAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://api.scrapingdog.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{
		Query: "test",
		Type:  SearchTypeImages,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertResultCount(t, resp, 1)
	if resp.Results[0].ImageURL != "https://example.com/img.jpg" {
		t.Errorf("unexpected imageURL: %s", resp.Results[0].ImageURL)
	}
}

func TestScrapingDogErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]interface{}{"error": "Invalid key"})
	}))
	defer server.Close()

	adapter := &ScrapingDogAdapter{
		apiKey: "bad",
		client: newTestClient(server.URL, "https://api.scrapingdog.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 401)
}

// ---------------------------------------------------------------------------
// BrightData
// ---------------------------------------------------------------------------

func TestBrightDataResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertHeader(t, r, "Authorization", "Bearer bd-key")

		writeJSON(w, map[string]interface{}{
			"organic": []interface{}{
				map[string]interface{}{
					"title":        "BD Result",
					"link":         "https://example.com",
					"description":  "BD desc",
					"display_link": "example.com",
				},
			},
			"knowledge_panel": map[string]interface{}{
				"title":       "BD KP",
				"description": "BD KP desc",
			},
			"people_also_ask": []interface{}{
				map[string]interface{}{"question": "BD PAA?"},
			},
			"related_searches": []interface{}{
				map[string]interface{}{"query": "bd related"},
			},
		})
	}))
	defer server.Close()

	adapter := &BrightDataAdapter{
		apiKey: "bd-key",
		client: newTestClient(server.URL, "https://api.brightdata.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "brightdata")
	assertResultCount(t, resp, 1)
	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
	assertPAACount(t, resp, 1)
	assertRelatedSearchCount(t, resp, 1)
}

func TestBrightDataErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		writeJSON(w, map[string]interface{}{"message": "Forbidden"})
	}))
	defer server.Close()

	adapter := &BrightDataAdapter{
		apiKey: "bad",
		client: newTestClient(server.URL, "https://api.brightdata.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 403)
}

// ---------------------------------------------------------------------------
// SearchCans
// ---------------------------------------------------------------------------

func TestSearchCansResponseMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, http.MethodPost)
		assertHeader(t, r, "Authorization", "Bearer sc-key")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["s"] != "test query" {
			t.Errorf("expected s='test query', got '%v'", body["s"])
		}
		if body["t"] != "google" {
			t.Errorf("expected t=google, got %v", body["t"])
		}

		writeJSON(w, map[string]interface{}{
			"organic_results": []interface{}{
				map[string]interface{}{
					"title":          "SC Result",
					"link":           "https://example.com",
					"snippet":        "SC desc",
					"displayed_link": "example.com",
				},
			},
			"people_also_ask": []interface{}{
				map[string]interface{}{"question": "SC PAA?"},
			},
			"knowledge_panel": map[string]interface{}{
				"title": "SC KP",
			},
		})
	}))
	defer server.Close()

	adapter := &SearchCansAdapter{
		apiKey: "sc-key",
		client: newTestClient(server.URL, "https://www.searchcans.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test query"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertProvider(t, resp, "searchcans")
	assertResultCount(t, resp, 1)
	assertPAACount(t, resp, 1)
	if resp.KnowledgePanel == nil {
		t.Fatal("expected knowledge panel")
	}
}

func TestSearchCansErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]interface{}{"error": "Unauthorized"})
	}))
	defer server.Close()

	adapter := &SearchCansAdapter{
		apiKey: "bad",
		client: newTestClient(server.URL, "https://www.searchcans.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 401)
}

// ---------------------------------------------------------------------------
// SupportsType
// ---------------------------------------------------------------------------

func TestSupportsType(t *testing.T) {
	tests := []struct {
		name     string
		adapter  SearchAdapter
		typ      SearchType
		expected bool
	}{
		{"serper web", NewSerperAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"serper images", NewSerperAdapter("key", http.DefaultClient), SearchTypeImages, true},
		{"serper news", NewSerperAdapter("key", http.DefaultClient), SearchTypeNews, true},
		{"serper videos", NewSerperAdapter("key", http.DefaultClient), SearchTypeVideos, true},
		{"serpapi web", NewSerpAPIAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"serpapi images", NewSerpAPIAdapter("key", http.DefaultClient), SearchTypeImages, true},
		{"google web", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeWeb, true},
		{"google images", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeImages, true},
		{"google news", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeNews, false},
		{"google videos", NewGoogleAdapter("key", "cx", http.DefaultClient), SearchTypeVideos, false},
		{"bing web", NewBingAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"bing videos", NewBingAdapter("key", http.DefaultClient), SearchTypeVideos, true},
		{"brave web", NewBraveAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"brave videos", NewBraveAdapter("key", http.DefaultClient), SearchTypeVideos, true},
		{"dataforseo web", NewDataForSeoAdapter("l", "p", http.DefaultClient), SearchTypeWeb, true},
		{"dataforseo news", NewDataForSeoAdapter("l", "p", http.DefaultClient), SearchTypeNews, true},
		{"dataforseo images", NewDataForSeoAdapter("l", "p", http.DefaultClient), SearchTypeImages, false},
		{"dataforseo videos", NewDataForSeoAdapter("l", "p", http.DefaultClient), SearchTypeVideos, false},
		{"searchapi web", NewSearchAPIAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"valueserp web", NewValueSerpAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"scrapingdog web", NewScrapingDogAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"scrapingdog images", NewScrapingDogAdapter("key", http.DefaultClient), SearchTypeImages, true},
		{"scrapingdog news", NewScrapingDogAdapter("key", http.DefaultClient), SearchTypeNews, true},
		{"scrapingdog videos", NewScrapingDogAdapter("key", http.DefaultClient), SearchTypeVideos, false},
		{"brightdata web", NewBrightDataAdapter("key", http.DefaultClient), SearchTypeWeb, true},
		{"brightdata videos", NewBrightDataAdapter("key", http.DefaultClient), SearchTypeVideos, true},
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

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestSerperContextCancellation(t *testing.T) {
	adapter := &SerperAdapter{
		apiKey: "key",
		client: http.DefaultClient,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before making request

	_, err := adapter.Search(ctx, SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestBingContextCancellation(t *testing.T) {
	adapter := &BingAdapter{
		apiKey: "key",
		client: http.DefaultClient,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := adapter.Search(ctx, SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// ---------------------------------------------------------------------------
// Provider Name()
// ---------------------------------------------------------------------------

func TestProviderNames(t *testing.T) {
	tests := []struct {
		adapter  SearchAdapter
		expected string
	}{
		{NewSerperAdapter("k", http.DefaultClient), "serper"},
		{NewSerpAPIAdapter("k", http.DefaultClient), "serpapi"},
		{NewGoogleAdapter("k", "cx", http.DefaultClient), "google"},
		{NewBingAdapter("k", http.DefaultClient), "bing"},
		{NewBraveAdapter("k", http.DefaultClient), "brave"},
		{NewDataForSeoAdapter("l", "p", http.DefaultClient), "dataforseo"},
		{NewSearchAPIAdapter("k", http.DefaultClient), "searchapi"},
		{NewValueSerpAdapter("k", http.DefaultClient), "valueserp"},
		{NewScrapingDogAdapter("k", http.DefaultClient), "scrapingdog"},
		{NewBrightDataAdapter("k", http.DefaultClient), "brightdata"},
		{NewSearchCansAdapter("k", http.DefaultClient), "searchcans"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if tc.adapter.Name() != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, tc.adapter.Name())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Network error handling
// ---------------------------------------------------------------------------

func TestSerperNetworkError(t *testing.T) {
	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient("http://127.0.0.1:1", "https://google.serper.dev"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestBingNetworkError(t *testing.T) {
	adapter := &BingAdapter{
		apiKey: "key",
		client: newTestClient("http://127.0.0.1:1", "https://api.bing.microsoft.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestBraveNetworkError(t *testing.T) {
	adapter := &BraveAdapter{
		apiKey: "key",
		client: newTestClient("http://127.0.0.1:1", "https://api.search.brave.com"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

// ---------------------------------------------------------------------------
// Empty response handling
// ---------------------------------------------------------------------------

func TestSerperEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"organic": []interface{}{}})
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestGoogleEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{})
	}))
	defer server.Close()

	adapter := &GoogleAdapter{
		apiKey:   "key",
		engineID: "cx",
		client:   newTestClient(server.URL, "https://www.googleapis.com"),
	}

	resp, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

// ---------------------------------------------------------------------------
// Malformed JSON
// ---------------------------------------------------------------------------

func TestSerperMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error from malformed JSON")
	}
}

func TestSerperMalformedJSONWithErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	adapter := &SerperAdapter{
		apiKey: "key",
		client: newTestClient(server.URL, "https://google.serper.dev"),
	}

	_, err := adapter.Search(context.Background(), SearchRequest{Query: "test"})
	assertAnySerpError(t, err, 500)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func serperMockWebResponse() map[string]interface{} {
	return map[string]interface{}{
		"organic": []interface{}{
			map[string]interface{}{
				"title": "Go Programming Language", "link": "https://go.dev",
				"snippet": "The Go programming language", "domain": "go.dev", "date": "2024-01-01",
			},
			map[string]interface{}{
				"title": "Go Tutorial", "link": "https://go.dev/tour",
				"snippet": "A tour of Go", "domain": "go.dev",
			},
		},
		"searchParameters": map[string]interface{}{"timeTaken": 0.35},
		"relatedSearches": []interface{}{
			map[string]interface{}{"query": "golang tutorial"},
			map[string]interface{}{"query": "go vs rust"},
		},
		"peopleAlsoAsk": []interface{}{
			map[string]interface{}{
				"question": "Is Go a good language?", "snippet": "Yes, Go is great.",
				"title": "Go Language Review", "link": "https://example.com/review",
			},
		},
		"knowledgeGraph": map[string]interface{}{
			"title": "Go", "type": "Programming Language",
			"description": "Go is a programming language",
			"descriptionSource": "Wikipedia", "descriptionLink": "https://en.wikipedia.org/wiki/Go",
			"imageUrl":   "https://go.dev/logo.png",
			"attributes": map[string]interface{}{"Developer": "Google"},
		},
		"answerBox": map[string]interface{}{
			"snippet": "Go is a statically typed language",
			"title": "About Go", "link": "https://go.dev/about",
		},
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func assertHeader(t *testing.T, r *http.Request, key, expected string) {
	t.Helper()
	if got := r.Header.Get(key); got != expected {
		t.Errorf("expected header %s=%q, got %q", key, expected, got)
	}
}

func assertMethod(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.Method != expected {
		t.Errorf("expected method %s, got %s", expected, r.Method)
	}
}

func assertPathSuffix(t *testing.T, r *http.Request, suffix string) {
	t.Helper()
	if !strings.HasSuffix(r.URL.Path, suffix) {
		t.Errorf("expected path suffix %q, got path %q", suffix, r.URL.Path)
	}
}

func assertProvider(t *testing.T, resp *SearchResponse, expected string) {
	t.Helper()
	if resp.Provider != expected {
		t.Errorf("expected provider %q, got %q", expected, resp.Provider)
	}
}

func assertQuery(t *testing.T, resp *SearchResponse, expected string) {
	t.Helper()
	if resp.Query != expected {
		t.Errorf("expected query %q, got %q", expected, resp.Query)
	}
}

func assertResultCount(t *testing.T, resp *SearchResponse, expected int) {
	t.Helper()
	if len(resp.Results) != expected {
		t.Fatalf("expected %d results, got %d", expected, len(resp.Results))
	}
}

func assertFirstResult(t *testing.T, resp *SearchResponse, title, url string) {
	t.Helper()
	if resp.Results[0].Title != title {
		t.Errorf("expected title %q, got %q", title, resp.Results[0].Title)
	}
	if resp.Results[0].URL != url {
		t.Errorf("expected URL %q, got %q", url, resp.Results[0].URL)
	}
}

func assertRelatedSearchCount(t *testing.T, resp *SearchResponse, expected int) {
	t.Helper()
	if len(resp.RelatedSearches) != expected {
		t.Errorf("expected %d related searches, got %d", expected, len(resp.RelatedSearches))
	}
}

func assertPAACount(t *testing.T, resp *SearchResponse, expected int) {
	t.Helper()
	if len(resp.PeopleAlsoAsk) != expected {
		t.Errorf("expected %d PAA, got %d", expected, len(resp.PeopleAlsoAsk))
	}
}

func assertAnySerpError(t *testing.T, err error, expectedCode int) *AnySerpError {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
		return nil
	}
	asErr, ok := err.(*AnySerpError)
	if !ok {
		t.Fatalf("expected *AnySerpError, got %T: %v", err, err)
		return nil
	}
	if asErr.Code != expectedCode {
		t.Errorf("expected error code %d, got %d", expectedCode, asErr.Code)
	}
	return asErr
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
