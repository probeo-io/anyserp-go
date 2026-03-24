package anyserp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var valueSerpSearchTypeMap = map[SearchType]string{
	SearchTypeWeb:    "web",
	SearchTypeImages: "images",
	SearchTypeNews:   "news",
	SearchTypeVideos: "videos",
}

var valueSerpTimePeriodMap = map[DateRange]string{
	DateRangeDay:   "last_day",
	DateRangeWeek:  "last_week",
	DateRangeMonth: "last_month",
	DateRangeYear:  "last_year",
}

// ValueSerpAdapter implements SearchAdapter for the ValueSERP API.
type ValueSerpAdapter struct {
	apiKey string
	client *http.Client
}

// NewValueSerpAdapter creates a new ValueSerpAdapter.
func NewValueSerpAdapter(apiKey string, client *http.Client) *ValueSerpAdapter {
	return &ValueSerpAdapter{apiKey: apiKey, client: client}
}

func (a *ValueSerpAdapter) Name() string { return "valueserp" }

func (a *ValueSerpAdapter) SupportsType(_ SearchType) bool { return true }

func (a *ValueSerpAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}

	params := url.Values{}
	params.Set("api_key", a.apiKey)
	params.Set("output", "json")
	params.Set("q", request.Query)
	params.Set("search_type", valueSerpSearchTypeMap[searchType])

	if request.Num > 0 {
		params.Set("num", strconv.Itoa(request.Num))
	}
	if request.Page > 1 {
		params.Set("page", strconv.Itoa(request.Page))
	}
	if request.Country != "" {
		params.Set("gl", request.Country)
	}
	if request.Language != "" {
		params.Set("hl", request.Language)
	}
	if request.Safe {
		params.Set("safe", "active")
	}
	if request.DateRange != "" {
		if v, ok := valueSerpTimePeriodMap[request.DateRange]; ok {
			params.Set("time_period", v)
		}
	}

	data, err := a.makeRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	switch searchType {
	case SearchTypeWeb:
		for i, r := range jsonArray(data, "organic_results") {
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Domain:        jsonStr(r, "domain"),
				DatePublished: jsonStr(r, "date"),
			})
		}
	case SearchTypeImages:
		for i, r := range jsonArray(data, "image_results") {
			imageURL := jsonStr(r, "original")
			if imageURL == "" {
				imageURL = jsonStr(r, "image")
			}
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         jsonStr(r, "link"),
				Description: jsonStr(r, "title"),
				ImageURL:    imageURL,
				ImageWidth:  jsonInt(r, "original_width"),
				ImageHeight: jsonInt(r, "original_height"),
				Thumbnail:   jsonStr(r, "thumbnail"),
				Source:      jsonStr(r, "source"),
			})
		}
	case SearchTypeNews:
		for i, r := range jsonArray(data, "news_results") {
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Source:        jsonStr(r, "source"),
				DatePublished: jsonStr(r, "date"),
				Thumbnail:     jsonStr(r, "thumbnail"),
			})
		}
	case SearchTypeVideos:
		for i, r := range jsonArray(data, "video_results") {
			desc := jsonStr(r, "snippet")
			if desc == "" {
				desc = jsonStr(r, "description")
			}
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   desc,
				Duration:      jsonStr(r, "duration"),
				Channel:       jsonStr(r, "channel"),
				Thumbnail:     jsonStr(r, "thumbnail"),
				DatePublished: jsonStr(r, "date"),
			})
		}
	}

	resp := &SearchResponse{
		Provider: "valueserp",
		Query:    request.Query,
		Results:  results,
	}

	if si := jsonObj(data, "search_information"); si != nil {
		resp.TotalResults = jsonInt(si, "total_results")
		if ttStr := jsonStr(si, "time_taken_displayed"); ttStr != "" {
			if tt, err := strconv.ParseFloat(ttStr, 64); err == nil {
				resp.SearchTime = tt * 1000
			}
		}
	}

	if rs := jsonArray(data, "related_searches"); len(rs) > 0 {
		for _, r := range rs {
			if q := jsonStr(r, "query"); q != "" {
				resp.RelatedSearches = append(resp.RelatedSearches, q)
			}
		}
	}

	if paa := jsonArray(data, "people_also_ask"); len(paa) > 0 {
		for _, q := range paa {
			resp.PeopleAlsoAsk = append(resp.PeopleAlsoAsk, PeopleAlsoAsk{
				Question: jsonStr(q, "question"),
				Snippet:  jsonStr(q, "snippet"),
				Title:    jsonStr(q, "title"),
				URL:      jsonStr(q, "link"),
			})
		}
	}

	if kg := jsonObj(data, "knowledge_graph"); kg != nil {
		kp := &KnowledgePanel{
			Title:       jsonStr(kg, "title"),
			Type:        jsonStr(kg, "type"),
			Description: jsonStr(kg, "description"),
			ImageURL:    jsonStr(kg, "image"),
		}
		if src := jsonObj(kg, "source"); src != nil {
			kp.Source = jsonStr(src, "name")
			kp.SourceURL = jsonStr(src, "link")
		}
		resp.KnowledgePanel = kp
	}

	if ab := jsonObj(data, "answer_box"); ab != nil {
		snippet := jsonStr(ab, "snippet")
		if snippet == "" {
			snippet = jsonStr(ab, "answer")
		}
		resp.AnswerBox = &AnswerBox{
			Snippet: snippet,
			Title:   jsonStr(ab, "title"),
			URL:     jsonStr(ab, "link"),
		}
	}

	return resp, nil
}

func (a *ValueSerpAdapter) makeRequest(ctx context.Context, params url.Values) (map[string]interface{}, error) {
	u := "https://api.valueserp.com/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "valueserp"})
		}
		return nil, fmt.Errorf("valueserp: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "error")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "valueserp", "raw": result})
	}

	return result, nil
}
