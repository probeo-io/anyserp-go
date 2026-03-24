package anyserp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var serpAPIEngineMap = map[SearchType]string{
	SearchTypeWeb:    "google",
	SearchTypeImages: "google_images",
	SearchTypeNews:   "google_news",
	SearchTypeVideos: "google_videos",
}

var serpAPIDateMap = map[DateRange]string{
	DateRangeDay:   "qdr:d",
	DateRangeWeek:  "qdr:w",
	DateRangeMonth: "qdr:m",
	DateRangeYear:  "qdr:y",
}

// SerpAPIAdapter implements SearchAdapter for the SerpAPI service.
type SerpAPIAdapter struct {
	apiKey string
	client *http.Client
}

// NewSerpAPIAdapter creates a new SerpAPIAdapter.
func NewSerpAPIAdapter(apiKey string, client *http.Client) *SerpAPIAdapter {
	return &SerpAPIAdapter{apiKey: apiKey, client: client}
}

func (a *SerpAPIAdapter) Name() string { return "serpapi" }

func (a *SerpAPIAdapter) SupportsType(_ SearchType) bool { return true }

func (a *SerpAPIAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}

	params := url.Values{}
	params.Set("engine", serpAPIEngineMap[searchType])
	params.Set("q", request.Query)
	params.Set("api_key", a.apiKey)
	params.Set("output", "json")

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
		if v, ok := serpAPIDateMap[request.DateRange]; ok {
			params.Set("tbs", v)
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
			pos := jsonInt(r, "position")
			if pos == 0 {
				pos = i + 1
			}
			results = append(results, SearchResult{
				Position:      pos,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Domain:        jsonStr(r, "displayed_link"),
				DatePublished: jsonStr(r, "date"),
				Thumbnail:     jsonStr(r, "thumbnail"),
			})
		}
	case SearchTypeImages:
		for i, r := range jsonArray(data, "images_results") {
			pos := jsonInt(r, "position")
			if pos == 0 {
				pos = i + 1
			}
			results = append(results, SearchResult{
				Position:    pos,
				Title:       jsonStr(r, "title"),
				URL:         jsonStr(r, "link"),
				Description: jsonStr(r, "snippet"),
				ImageURL:    jsonStr(r, "original"),
				ImageWidth:  jsonInt(r, "original_width"),
				ImageHeight: jsonInt(r, "original_height"),
				Thumbnail:   jsonStr(r, "thumbnail"),
				Source:      jsonStr(r, "source"),
			})
		}
	case SearchTypeNews:
		for i, r := range jsonArray(data, "news_results") {
			pos := jsonInt(r, "position")
			if pos == 0 {
				pos = i + 1
			}
			source := jsonStr(r, "source")
			if source == "" {
				if srcObj := jsonObj(r, "source"); srcObj != nil {
					source = jsonStr(srcObj, "name")
				}
			}
			results = append(results, SearchResult{
				Position:      pos,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Source:        source,
				DatePublished: jsonStr(r, "date"),
				Thumbnail:     jsonStr(r, "thumbnail"),
			})
		}
	case SearchTypeVideos:
		for i, r := range jsonArray(data, "video_results") {
			pos := jsonInt(r, "position")
			if pos == 0 {
				pos = i + 1
			}
			channel := ""
			if ch := jsonObj(r, "channel"); ch != nil {
				channel = jsonStr(ch, "name")
			}
			thumbnail := ""
			if th := jsonObj(r, "thumbnail"); th != nil {
				thumbnail = jsonStr(th, "static")
			}
			results = append(results, SearchResult{
				Position:      pos,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Duration:      jsonStr(r, "duration"),
				Channel:       channel,
				Thumbnail:     thumbnail,
				DatePublished: jsonStr(r, "date"),
			})
		}
	}

	resp := &SearchResponse{
		Provider: "serpapi",
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

	if rq := jsonArray(data, "related_questions"); len(rq) > 0 {
		for _, q := range rq {
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
		}
		if src := jsonObj(kg, "source"); src != nil {
			kp.Source = jsonStr(src, "name")
			kp.SourceURL = jsonStr(src, "link")
		}
		if hi := jsonArray(kg, "header_images"); len(hi) > 0 {
			kp.ImageURL = jsonStr(hi[0], "image")
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

func (a *SerpAPIAdapter) makeRequest(ctx context.Context, params url.Values) (map[string]interface{}, error) {
	u := "https://serpapi.com/search.json?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "serpapi"})
		}
		return nil, fmt.Errorf("serpapi: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "error")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "serpapi", "raw": result})
	}

	return result, nil
}
