package anyserp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var brightDataTBMMap = map[SearchType]string{
	SearchTypeImages: "isch",
	SearchTypeNews:   "nws",
	SearchTypeVideos: "vid",
}

// BrightDataAdapter implements SearchAdapter for the Bright Data SERP API.
type BrightDataAdapter struct {
	apiKey string
	client *http.Client
}

// NewBrightDataAdapter creates a new BrightDataAdapter.
func NewBrightDataAdapter(apiKey string, client *http.Client) *BrightDataAdapter {
	return &BrightDataAdapter{apiKey: apiKey, client: client}
}

func (a *BrightDataAdapter) Name() string { return "brightdata" }

func (a *BrightDataAdapter) SupportsType(_ SearchType) bool { return true }

func (a *BrightDataAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}

	searchURL := a.buildSearchURL(request, searchType)
	data, err := a.makeRequest(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	switch searchType {
	case SearchTypeWeb:
		for i, r := range jsonArray(data, "organic") {
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         jsonStr(r, "link"),
				Description: jsonStr(r, "description"),
				Domain:      jsonStr(r, "display_link"),
			})
		}
	case SearchTypeImages:
		arr := jsonArray(data, "organic")
		if len(arr) == 0 {
			arr = jsonArray(data, "images")
		}
		for i, r := range arr {
			desc := jsonStr(r, "description")
			if desc == "" {
				desc = jsonStr(r, "title")
			}
			imageURL := jsonStr(r, "original")
			if imageURL == "" {
				imageURL = jsonStr(r, "link")
			}
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         jsonStr(r, "link"),
				Description: desc,
				ImageURL:    imageURL,
				Thumbnail:   jsonStr(r, "thumbnail"),
				Source:      jsonStr(r, "display_link"),
			})
		}
	case SearchTypeNews:
		arr := jsonArray(data, "organic")
		if len(arr) == 0 {
			arr = jsonArray(data, "news")
		}
		for i, r := range arr {
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "description"),
				Source:        jsonStr(r, "display_link"),
				DatePublished: jsonStr(r, "date"),
			})
		}
	case SearchTypeVideos:
		arr := jsonArray(data, "organic")
		if len(arr) == 0 {
			arr = jsonArray(data, "videos")
		}
		for i, r := range arr {
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         jsonStr(r, "link"),
				Description: jsonStr(r, "description"),
				Thumbnail:   jsonStr(r, "thumbnail"),
				Duration:    jsonStr(r, "duration"),
			})
		}
	}

	resp := &SearchResponse{
		Provider: "brightdata",
		Query:    request.Query,
		Results:  results,
	}

	if kp := jsonObj(data, "knowledge_panel"); kp != nil {
		resp.KnowledgePanel = &KnowledgePanel{
			Title:       jsonStr(kp, "title"),
			Type:        jsonStr(kp, "type"),
			Description: jsonStr(kp, "description"),
			ImageURL:    jsonStr(kp, "image"),
		}
	}

	if paa := jsonArray(data, "people_also_ask"); len(paa) > 0 {
		for _, q := range paa {
			u := jsonStr(q, "link")
			if u == "" {
				u = jsonStr(q, "url")
			}
			resp.PeopleAlsoAsk = append(resp.PeopleAlsoAsk, PeopleAlsoAsk{
				Question: jsonStr(q, "question"),
				Snippet:  jsonStr(q, "snippet"),
				Title:    jsonStr(q, "title"),
				URL:      u,
			})
		}
	}

	if rs := jsonArray(data, "related_searches"); len(rs) > 0 {
		for _, r := range rs {
			query := jsonStr(r, "query")
			if query == "" {
				query = jsonStr(r, "title")
			}
			// Handle case where related_searches items are strings
			if query == "" {
				if s, ok := r["_str"]; ok {
					if str, ok := s.(string); ok {
						query = str
					}
				}
			}
			if query != "" {
				resp.RelatedSearches = append(resp.RelatedSearches, query)
			}
		}
	}

	return resp, nil
}

func (a *BrightDataAdapter) buildSearchURL(request SearchRequest, searchType SearchType) string {
	params := url.Values{}
	params.Set("q", request.Query)

	if tbm, ok := brightDataTBMMap[searchType]; ok {
		params.Set("tbm", tbm)
	}

	if request.Country != "" {
		params.Set("gl", request.Country)
	}
	if request.Language != "" {
		params.Set("hl", request.Language)
	}
	if request.Num > 0 {
		params.Set("num", strconv.Itoa(request.Num))
	}
	if request.Page > 1 {
		num := request.Num
		if num == 0 {
			num = 10
		}
		params.Set("start", strconv.Itoa((request.Page-1)*num))
	}
	if request.Safe {
		params.Set("safe", "active")
	}

	params.Set("brd_json", "1")

	return "https://www.google.com/search?" + params.Encode()
}

func (a *BrightDataAdapter) makeRequest(ctx context.Context, searchURL string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"zone":   "serp",
		"url":    searchURL,
		"format": "raw",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brightdata.com/request", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "brightdata"})
		}
		return nil, fmt.Errorf("brightdata: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "message")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "brightdata", "raw": result})
	}

	return result, nil
}
