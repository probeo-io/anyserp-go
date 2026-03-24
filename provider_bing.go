package anyserp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var bingTypeEndpoints = map[SearchType]string{
	SearchTypeWeb:    "/search",
	SearchTypeImages: "/images/search",
	SearchTypeNews:   "/news/search",
	SearchTypeVideos: "/videos/search",
}

var bingFreshnessMap = map[DateRange]string{
	DateRangeDay:   "Day",
	DateRangeWeek:  "Week",
	DateRangeMonth: "Month",
}

// BingAdapter implements SearchAdapter for the Bing Web Search API.
type BingAdapter struct {
	apiKey string
	client *http.Client
}

// NewBingAdapter creates a new BingAdapter.
func NewBingAdapter(apiKey string, client *http.Client) *BingAdapter {
	return &BingAdapter{apiKey: apiKey, client: client}
}

func (a *BingAdapter) Name() string { return "bing" }

func (a *BingAdapter) SupportsType(_ SearchType) bool { return true }

func (a *BingAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}
	endpoint := bingTypeEndpoints[searchType]

	params := url.Values{}
	params.Set("q", request.Query)

	if request.Num > 0 {
		params.Set("count", strconv.Itoa(request.Num))
	}
	if request.Page > 1 {
		num := request.Num
		if num == 0 {
			num = 10
		}
		params.Set("offset", strconv.Itoa((request.Page-1)*num))
	}
	if request.Country != "" {
		params.Set("cc", request.Country)
	}
	if request.Language != "" {
		params.Set("setLang", request.Language)
	}
	if request.Safe {
		params.Set("safeSearch", "Strict")
	}
	if request.DateRange != "" {
		if v, ok := bingFreshnessMap[request.DateRange]; ok {
			params.Set("freshness", v)
		}
	}

	data, err := a.makeRequest(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	switch searchType {
	case SearchTypeWeb:
		if wp := jsonObj(data, "webPages"); wp != nil {
			for i, r := range jsonArray(wp, "value") {
				domain := ""
				if u := jsonStr(r, "url"); u != "" {
					if parsed, err := url.Parse(u); err == nil {
						domain = parsed.Hostname()
					}
				}
				results = append(results, SearchResult{
					Position:      i + 1,
					Title:         jsonStr(r, "name"),
					URL:           jsonStr(r, "url"),
					Description:   jsonStr(r, "snippet"),
					Domain:        domain,
					DatePublished: jsonStr(r, "dateLastCrawled"),
				})
			}
		}
	case SearchTypeImages:
		for i, r := range jsonArray(data, "value") {
			domain := ""
			if u := jsonStr(r, "hostPageUrl"); u != "" {
				if parsed, err := url.Parse(u); err == nil {
					domain = parsed.Hostname()
				}
			}
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "name"),
				URL:         jsonStr(r, "hostPageUrl"),
				Description: jsonStr(r, "name"),
				ImageURL:    jsonStr(r, "contentUrl"),
				ImageWidth:  jsonInt(r, "width"),
				ImageHeight: jsonInt(r, "height"),
				Thumbnail:   jsonStr(r, "thumbnailUrl"),
				Domain:      domain,
			})
		}
	case SearchTypeNews:
		for i, r := range jsonArray(data, "value") {
			source := ""
			if providers := jsonArray(r, "provider"); len(providers) > 0 {
				source = jsonStr(providers[0], "name")
			}
			thumbnail := ""
			if img := jsonObj(r, "image"); img != nil {
				if th := jsonObj(img, "thumbnail"); th != nil {
					thumbnail = jsonStr(th, "contentUrl")
				}
			}
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "name"),
				URL:           jsonStr(r, "url"),
				Description:   jsonStr(r, "description"),
				Source:        source,
				DatePublished: jsonStr(r, "datePublished"),
				Thumbnail:     thumbnail,
			})
		}
	case SearchTypeVideos:
		for i, r := range jsonArray(data, "value") {
			channel := ""
			if cr := jsonObj(r, "creator"); cr != nil {
				channel = jsonStr(cr, "name")
			}
			u := jsonStr(r, "contentUrl")
			if u == "" {
				u = jsonStr(r, "hostPageUrl")
			}
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "name"),
				URL:           u,
				Description:   jsonStr(r, "description"),
				Duration:      jsonStr(r, "duration"),
				Channel:       channel,
				Thumbnail:     jsonStr(r, "thumbnailUrl"),
				DatePublished: jsonStr(r, "datePublished"),
			})
		}
	}

	resp := &SearchResponse{
		Provider: "bing",
		Query:    request.Query,
		Results:  results,
	}

	if wp := jsonObj(data, "webPages"); wp != nil {
		resp.TotalResults = jsonInt(wp, "totalEstimatedMatches")
	}
	if resp.TotalResults == 0 {
		resp.TotalResults = jsonInt(data, "totalEstimatedMatches")
	}

	return resp, nil
}

func (a *BingAdapter) makeRequest(ctx context.Context, endpoint string, params url.Values) (map[string]interface{}, error) {
	u := "https://api.bing.microsoft.com/v7.0" + endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "bing"})
		}
		return nil, fmt.Errorf("bing: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := resp.Status
		if errObj := jsonObj(result, "error"); errObj != nil {
			if m := jsonStr(errObj, "message"); m != "" {
				msg = m
			}
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "bing", "raw": result})
	}

	return result, nil
}
