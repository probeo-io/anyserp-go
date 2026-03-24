package anyserp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var serperTypeEndpoints = map[SearchType]string{
	SearchTypeWeb:    "/search",
	SearchTypeImages: "/images",
	SearchTypeNews:   "/news",
	SearchTypeVideos: "/videos",
}

var serperDateMap = map[DateRange]string{
	DateRangeDay:   "qdr:d",
	DateRangeWeek:  "qdr:w",
	DateRangeMonth: "qdr:m",
	DateRangeYear:  "qdr:y",
}

// SerperAdapter implements SearchAdapter for the Serper API.
type SerperAdapter struct {
	apiKey string
	client *http.Client
}

// NewSerperAdapter creates a new SerperAdapter.
func NewSerperAdapter(apiKey string, client *http.Client) *SerperAdapter {
	return &SerperAdapter{apiKey: apiKey, client: client}
}

func (a *SerperAdapter) Name() string { return "serper" }

func (a *SerperAdapter) SupportsType(_ SearchType) bool { return true }

func (a *SerperAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}
	endpoint := serperTypeEndpoints[searchType]

	body := map[string]interface{}{"q": request.Query}
	if request.Num > 0 {
		body["num"] = request.Num
	}
	if request.Page > 1 {
		body["page"] = request.Page
	}
	if request.Country != "" {
		body["gl"] = request.Country
	}
	if request.Language != "" {
		body["hl"] = request.Language
	}
	if request.DateRange != "" {
		if v, ok := serperDateMap[request.DateRange]; ok {
			body["tbs"] = v
		}
	}

	data, err := a.makeRequest(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	switch searchType {
	case SearchTypeWeb:
		for i, r := range jsonArray(data, "organic") {
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
		for i, r := range jsonArray(data, "images") {
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         jsonStr(r, "link"),
				Description: jsonStr(r, "snippet"),
				ImageURL:    jsonStr(r, "imageUrl"),
				ImageWidth:  jsonInt(r, "imageWidth"),
				ImageHeight: jsonInt(r, "imageHeight"),
				Domain:      jsonStr(r, "domain"),
				Thumbnail:   jsonStr(r, "thumbnailUrl"),
			})
		}
	case SearchTypeNews:
		for i, r := range jsonArray(data, "news") {
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Source:        jsonStr(r, "source"),
				DatePublished: jsonStr(r, "date"),
				Thumbnail:     jsonStr(r, "imageUrl"),
			})
		}
	case SearchTypeVideos:
		for i, r := range jsonArray(data, "videos") {
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Duration:      jsonStr(r, "duration"),
				Channel:       jsonStr(r, "channel"),
				Thumbnail:     jsonStr(r, "imageUrl"),
				DatePublished: jsonStr(r, "date"),
			})
		}
	}

	resp := &SearchResponse{
		Provider: "serper",
		Query:    request.Query,
		Results:  results,
	}

	if sp := jsonObj(data, "searchParameters"); sp != nil {
		if tt := jsonFloat(sp, "timeTaken"); tt > 0 {
			resp.SearchTime = tt * 1000
		}
	}

	if rs := jsonArray(data, "relatedSearches"); len(rs) > 0 {
		for _, r := range rs {
			if q := jsonStr(r, "query"); q != "" {
				resp.RelatedSearches = append(resp.RelatedSearches, q)
			}
		}
	}

	if paa := jsonArray(data, "peopleAlsoAsk"); len(paa) > 0 {
		for _, p := range paa {
			resp.PeopleAlsoAsk = append(resp.PeopleAlsoAsk, PeopleAlsoAsk{
				Question: jsonStr(p, "question"),
				Snippet:  jsonStr(p, "snippet"),
				Title:    jsonStr(p, "title"),
				URL:      jsonStr(p, "link"),
			})
		}
	}

	if kg := jsonObj(data, "knowledgeGraph"); kg != nil {
		kp := &KnowledgePanel{
			Title:       jsonStr(kg, "title"),
			Type:        jsonStr(kg, "type"),
			Description: jsonStr(kg, "description"),
			Source:      jsonStr(kg, "descriptionSource"),
			SourceURL:   jsonStr(kg, "descriptionLink"),
			ImageURL:    jsonStr(kg, "imageUrl"),
		}
		if attrs := jsonObj(kg, "attributes"); attrs != nil {
			kp.Attributes = make(map[string]string)
			for k, v := range attrs {
				if s, ok := v.(string); ok {
					kp.Attributes[k] = s
				}
			}
		}
		resp.KnowledgePanel = kp
	}

	if ab := jsonObj(data, "answerBox"); ab != nil {
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

func (a *SerperAdapter) makeRequest(ctx context.Context, endpoint string, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://google.serper.dev"+endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "serper"})
		}
		return nil, fmt.Errorf("serper: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "message")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "serper", "raw": result})
	}

	return result, nil
}
