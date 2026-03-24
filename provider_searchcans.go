package anyserp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SearchCansAdapter implements SearchAdapter for the SearchCans API.
type SearchCansAdapter struct {
	apiKey string
	client *http.Client
}

// NewSearchCansAdapter creates a new SearchCansAdapter.
func NewSearchCansAdapter(apiKey string, client *http.Client) *SearchCansAdapter {
	return &SearchCansAdapter{apiKey: apiKey, client: client}
}

func (a *SearchCansAdapter) Name() string { return "searchcans" }

func (a *SearchCansAdapter) SupportsType(t SearchType) bool {
	return t == SearchTypeWeb || t == SearchTypeNews
}

func (a *SearchCansAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	body := map[string]interface{}{
		"s": request.Query,
		"t": "google",
	}

	if request.Page > 0 {
		body["p"] = request.Page
	}
	if request.Country != "" {
		body["gl"] = request.Country
	}
	if request.Language != "" {
		body["hl"] = request.Language
	}

	data, err := a.makeRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	organic := jsonArray(data, "organic_results")
	if len(organic) == 0 {
		organic = jsonArray(data, "results")
	}

	for i, r := range organic {
		u := jsonStr(r, "link")
		if u == "" {
			u = jsonStr(r, "url")
		}
		desc := jsonStr(r, "snippet")
		if desc == "" {
			desc = jsonStr(r, "description")
		}
		domain := jsonStr(r, "displayed_link")
		if domain == "" {
			domain = jsonStr(r, "domain")
		}
		results = append(results, SearchResult{
			Position:      i + 1,
			Title:         jsonStr(r, "title"),
			URL:           u,
			Description:   desc,
			Domain:        domain,
			DatePublished: jsonStr(r, "date"),
		})
	}

	resp := &SearchResponse{
		Provider: "searchcans",
		Query:    request.Query,
		Results:  results,
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

	if kp := jsonObj(data, "knowledge_panel"); kp != nil {
		resp.KnowledgePanel = &KnowledgePanel{
			Title:       jsonStr(kp, "title"),
			Type:        jsonStr(kp, "type"),
			Description: jsonStr(kp, "description"),
		}
	}

	return resp, nil
}

func (a *SearchCansAdapter) makeRequest(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.searchcans.com/api/search", bytes.NewReader(jsonBody))
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
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "searchcans"})
		}
		return nil, fmt.Errorf("searchcans: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "error")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "searchcans", "raw": result})
	}

	return result, nil
}
