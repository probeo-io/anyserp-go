package anyserp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var searchAPIEngineMap = map[SearchType]string{
	SearchTypeWeb:    "google",
	SearchTypeImages: "google_images",
	SearchTypeNews:   "google_news",
	SearchTypeVideos: "google_videos",
}

var searchAPITimePeriodMap = map[DateRange]string{
	DateRangeDay:   "last_day",
	DateRangeWeek:  "last_week",
	DateRangeMonth: "last_month",
	DateRangeYear:  "last_year",
}

// SearchAPIAdapter implements SearchAdapter for the SearchAPI.io service.
type SearchAPIAdapter struct {
	apiKey string
	client *http.Client
}

// NewSearchAPIAdapter creates a new SearchAPIAdapter.
func NewSearchAPIAdapter(apiKey string, client *http.Client) *SearchAPIAdapter {
	return &SearchAPIAdapter{apiKey: apiKey, client: client}
}

func (a *SearchAPIAdapter) Name() string { return "searchapi" }

func (a *SearchAPIAdapter) SupportsType(_ SearchType) bool { return true }

func (a *SearchAPIAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}

	params := url.Values{}
	params.Set("engine", searchAPIEngineMap[searchType])
	params.Set("q", request.Query)

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
		if v, ok := searchAPITimePeriodMap[request.DateRange]; ok {
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
			domain := ""
			if link := jsonStr(r, "link"); link != "" {
				if parsed, err := url.Parse(link); err == nil {
					domain = parsed.Hostname()
				}
			}
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "link"),
				Description:   jsonStr(r, "snippet"),
				Domain:        domain,
				DatePublished: jsonStr(r, "date"),
				Thumbnail:     jsonStr(r, "thumbnail"),
			})
		}
	case SearchTypeImages:
		arr := jsonArray(data, "images")
		if len(arr) == 0 {
			arr = jsonArray(data, "image_results")
		}
		for i, r := range arr {
			u := jsonStr(r, "link")
			if u == "" {
				u = jsonStr(r, "original")
			}
			desc := jsonStr(r, "snippet")
			if desc == "" {
				desc = jsonStr(r, "title")
			}
			imageURL := jsonStr(r, "original")
			if imageURL == "" {
				imageURL = jsonStr(r, "image")
			}
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         u,
				Description: desc,
				ImageURL:    imageURL,
				ImageWidth:  jsonInt(r, "original_width"),
				ImageHeight: jsonInt(r, "original_height"),
				Thumbnail:   jsonStr(r, "thumbnail"),
				Source:      jsonStr(r, "source"),
			})
		}
	case SearchTypeNews:
		arr := jsonArray(data, "news_results")
		if len(arr) == 0 {
			arr = jsonArray(data, "organic_results")
		}
		for i, r := range arr {
			source := jsonStr(r, "source")
			if source == "" {
				if srcObj := jsonObj(r, "source"); srcObj != nil {
					source = jsonStr(srcObj, "name")
				}
			}
			results = append(results, SearchResult{
				Position:      i + 1,
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
			channel := jsonStr(r, "channel")
			if channel == "" {
				if ch := jsonObj(r, "channel"); ch != nil {
					channel = jsonStr(ch, "name")
				}
			}
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
				Channel:       channel,
				Thumbnail:     jsonStr(r, "thumbnail"),
				DatePublished: jsonStr(r, "date"),
			})
		}
	}

	resp := &SearchResponse{
		Provider: "searchapi",
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

	// AI Overview
	if aiObj := jsonObj(data, "ai_overview"); aiObj != nil {
		pageToken := jsonStr(aiObj, "page_token")
		if pageToken != "" && request.IncludeAiOverview {
			aiParams := url.Values{}
			aiParams.Set("engine", "google_ai_overview")
			aiParams.Set("page_token", pageToken)
			aiData, err := a.makeRequest(ctx, aiParams)
			if err == nil {
				resp.AiOverview = mapAiOverview(aiData, pageToken)
			}
			// If AI overview fetch fails, don't fail the whole search
		} else if pageToken != "" {
			resp.AiOverview = &AiOverview{
				TextBlocks: []AiOverviewTextBlock{},
				References: []AiOverviewReference{},
				PageToken:  pageToken,
			}
		}
	}

	return resp, nil
}

func (a *SearchAPIAdapter) makeRequest(ctx context.Context, params url.Values) (map[string]interface{}, error) {
	u := "https://www.searchapi.io/api/v1/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "searchapi"})
		}
		return nil, fmt.Errorf("searchapi: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "error")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "searchapi", "raw": result})
	}

	return result, nil
}

func mapAiOverview(data map[string]interface{}, pageToken string) *AiOverview {
	var textBlocks []AiOverviewTextBlock
	for _, block := range jsonArray(data, "text_blocks") {
		textBlocks = append(textBlocks, mapTextBlock(block))
	}

	var references []AiOverviewReference
	for _, ref := range jsonArray(data, "reference_links") {
		references = append(references, AiOverviewReference{
			Index:     jsonInt(ref, "index"),
			Title:     jsonStr(ref, "title"),
			URL:       jsonStr(ref, "link"),
			Snippet:   jsonStr(ref, "snippet"),
			Date:      jsonStr(ref, "date"),
			Source:    jsonStr(ref, "source"),
			Thumbnail: jsonStr(ref, "thumbnail"),
		})
	}

	if textBlocks == nil {
		textBlocks = []AiOverviewTextBlock{}
	}
	if references == nil {
		references = []AiOverviewReference{}
	}

	return &AiOverview{
		Markdown:   jsonStr(data, "markdown"),
		TextBlocks: textBlocks,
		References: references,
		PageToken:  pageToken,
	}
}

func mapTextBlock(block map[string]interface{}) AiOverviewTextBlock {
	blockType := jsonStr(block, "type")
	if blockType == "" {
		blockType = "paragraph"
	}

	tb := AiOverviewTextBlock{
		Type:            blockType,
		Answer:          jsonStr(block, "answer"),
		AnswerHighlight: jsonStr(block, "answer_highlight"),
		Link:            jsonStr(block, "link"),
	}

	if ri := jsonIntArray(block, "reference_indexes"); len(ri) > 0 {
		tb.ReferenceIndexes = ri
	}

	if rs := jsonArray(block, "related_searches"); len(rs) > 0 {
		for _, r := range rs {
			tb.RelatedSearches = append(tb.RelatedSearches, RelatedSearch{
				Query: jsonStr(r, "query"),
				Link:  jsonStr(r, "link"),
			})
		}
	}

	if items := jsonArray(block, "items"); len(items) > 0 {
		for _, item := range items {
			tb.Items = append(tb.Items, mapTextBlock(item))
		}
	}

	if table := jsonObj(block, "table"); table != nil {
		t := &AiOverviewTable{}
		for _, h := range jsonStrArray(table, "headers") {
			t.Headers = append(t.Headers, h)
		}
		if rows, ok := table["rows"]; ok {
			if rowsArr, ok := rows.([]interface{}); ok {
				for _, row := range rowsArr {
					if rowArr, ok := row.([]interface{}); ok {
						var rowStrs []string
						for _, cell := range rowArr {
							if s, ok := cell.(string); ok {
								rowStrs = append(rowStrs, s)
							}
						}
						t.Rows = append(t.Rows, rowStrs)
					}
				}
			}
		}
		tb.Table = t
	}

	if blockType == "code_blocks" {
		tb.Language = jsonStr(block, "language")
		tb.Code = jsonStr(block, "code")
	}

	if blockType == "video" {
		tb.Video = &AiOverviewVideo{
			Title:    jsonStr(block, "title"),
			Link:     jsonStr(block, "link"),
			Duration: jsonStr(block, "duration"),
			Source:   jsonStr(block, "source"),
			Channel:  jsonStr(block, "channel"),
		}
	}

	return tb
}
