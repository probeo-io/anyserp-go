package anyserp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var googleDateMap = map[DateRange]string{
	DateRangeDay:   "d1",
	DateRangeWeek:  "w1",
	DateRangeMonth: "m1",
	DateRangeYear:  "y1",
}

// GoogleAdapter implements SearchAdapter for Google Custom Search Engine.
type GoogleAdapter struct {
	apiKey   string
	engineID string
	client   *http.Client
}

// NewGoogleAdapter creates a new GoogleAdapter.
func NewGoogleAdapter(apiKey, engineID string, client *http.Client) *GoogleAdapter {
	return &GoogleAdapter{apiKey: apiKey, engineID: engineID, client: client}
}

func (a *GoogleAdapter) Name() string { return "google" }

func (a *GoogleAdapter) SupportsType(t SearchType) bool {
	return t == SearchTypeWeb || t == SearchTypeImages
}

func (a *GoogleAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}

	params := url.Values{}
	params.Set("key", a.apiKey)
	params.Set("cx", a.engineID)
	params.Set("q", request.Query)

	if request.Num > 0 {
		num := request.Num
		if num > 10 {
			num = 10
		}
		params.Set("num", strconv.Itoa(num))
	}
	if request.Page > 1 {
		num := request.Num
		if num == 0 {
			num = 10
		}
		params.Set("start", strconv.Itoa((request.Page-1)*num+1))
	}
	if request.Country != "" {
		params.Set("gl", request.Country)
	}
	if request.Language != "" {
		params.Set("lr", "lang_"+request.Language)
	}
	if request.Safe {
		params.Set("safe", "active")
	}
	if request.DateRange != "" {
		if v, ok := googleDateMap[request.DateRange]; ok {
			params.Set("dateRestrict", v)
		}
	}
	if searchType == SearchTypeImages {
		params.Set("searchType", "image")
	}

	data, err := a.makeRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for i, item := range jsonArray(data, "items") {
		r := SearchResult{
			Position:    i + 1,
			Title:       jsonStr(item, "title"),
			URL:         jsonStr(item, "link"),
			Description: jsonStr(item, "snippet"),
			Domain:      jsonStr(item, "displayLink"),
		}

		if searchType == SearchTypeImages {
			if img := jsonObj(item, "image"); img != nil {
				r.ImageURL = jsonStr(item, "link")
				r.ImageWidth = jsonInt(img, "width")
				r.ImageHeight = jsonInt(img, "height")
				r.Thumbnail = jsonStr(img, "thumbnailLink")
			}
		}

		if pm := jsonObj(item, "pagemap"); pm != nil {
			if metas := jsonArray(pm, "metatags"); len(metas) > 0 {
				r.DatePublished = jsonStr(metas[0], "article:published_time")
			}
		}

		results = append(results, r)
	}

	resp := &SearchResponse{
		Provider: "google",
		Query:    request.Query,
		Results:  results,
	}

	if si := jsonObj(data, "searchInformation"); si != nil {
		if trStr := jsonStr(si, "totalResults"); trStr != "" {
			if tr, err := strconv.Atoi(trStr); err == nil {
				resp.TotalResults = tr
			}
		}
		if st := jsonFloat(si, "searchTime"); st > 0 {
			resp.SearchTime = st * 1000
		}
	}

	return resp, nil
}

func (a *GoogleAdapter) makeRequest(ctx context.Context, params url.Values) (map[string]interface{}, error) {
	u := "https://www.googleapis.com/customsearch/v1?" + params.Encode()

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
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "google"})
		}
		return nil, fmt.Errorf("google: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := resp.Status
		if errObj := jsonObj(result, "error"); errObj != nil {
			if m := jsonStr(errObj, "message"); m != "" {
				msg = m
			}
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "google", "raw": result})
	}

	return result, nil
}
