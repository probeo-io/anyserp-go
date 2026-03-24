package anyserp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var braveTypeEndpoints = map[SearchType]string{
	SearchTypeWeb:    "/web/search",
	SearchTypeImages: "/images/search",
	SearchTypeNews:   "/news/search",
	SearchTypeVideos: "/videos/search",
}

var braveFreshnessMap = map[DateRange]string{
	DateRangeDay:   "pd",
	DateRangeWeek:  "pw",
	DateRangeMonth: "pm",
	DateRangeYear:  "py",
}

// BraveAdapter implements SearchAdapter for the Brave Search API.
type BraveAdapter struct {
	apiKey string
	client *http.Client
}

// NewBraveAdapter creates a new BraveAdapter.
func NewBraveAdapter(apiKey string, client *http.Client) *BraveAdapter {
	return &BraveAdapter{apiKey: apiKey, client: client}
}

func (a *BraveAdapter) Name() string { return "brave" }

func (a *BraveAdapter) SupportsType(_ SearchType) bool { return true }

func (a *BraveAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}
	endpoint := braveTypeEndpoints[searchType]

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
		params.Set("country", request.Country)
	}
	if request.Language != "" {
		params.Set("search_lang", request.Language)
	}
	if request.Safe {
		params.Set("safesearch", "strict")
	}
	if request.DateRange != "" {
		if v, ok := braveFreshnessMap[request.DateRange]; ok {
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
		if web := jsonObj(data, "web"); web != nil {
			for i, r := range jsonArray(web, "results") {
				domain := ""
				if mu := jsonObj(r, "meta_url"); mu != nil {
					domain = jsonStr(mu, "hostname")
				}
				thumbnail := ""
				if th := jsonObj(r, "thumbnail"); th != nil {
					thumbnail = jsonStr(th, "src")
				}
				results = append(results, SearchResult{
					Position:      i + 1,
					Title:         jsonStr(r, "title"),
					URL:           jsonStr(r, "url"),
					Description:   jsonStr(r, "description"),
					Domain:        domain,
					DatePublished: jsonStr(r, "page_age"),
					Thumbnail:     thumbnail,
				})
			}
		}
	case SearchTypeImages:
		for i, r := range jsonArray(data, "results") {
			thumbnail := ""
			if th := jsonObj(r, "thumbnail"); th != nil {
				thumbnail = jsonStr(th, "src")
			}
			imageURL := ""
			width := 0
			height := 0
			if props := jsonObj(r, "properties"); props != nil {
				imageURL = jsonStr(props, "url")
				width = jsonInt(props, "width")
				height = jsonInt(props, "height")
			}
			desc := jsonStr(r, "description")
			if desc == "" {
				desc = jsonStr(r, "title")
			}
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         jsonStr(r, "url"),
				Description: desc,
				ImageURL:    imageURL,
				ImageWidth:  width,
				ImageHeight: height,
				Thumbnail:   thumbnail,
				Source:      jsonStr(r, "source"),
			})
		}
	case SearchTypeNews:
		for i, r := range jsonArray(data, "results") {
			source := ""
			if mu := jsonObj(r, "meta_url"); mu != nil {
				source = jsonStr(mu, "hostname")
			}
			thumbnail := ""
			if th := jsonObj(r, "thumbnail"); th != nil {
				thumbnail = jsonStr(th, "src")
			}
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "url"),
				Description:   jsonStr(r, "description"),
				Source:        source,
				DatePublished: jsonStr(r, "age"),
				Thumbnail:     thumbnail,
			})
		}
	case SearchTypeVideos:
		for i, r := range jsonArray(data, "results") {
			thumbnail := ""
			if th := jsonObj(r, "thumbnail"); th != nil {
				thumbnail = jsonStr(th, "src")
			}
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           jsonStr(r, "url"),
				Description:   jsonStr(r, "description"),
				Thumbnail:     thumbnail,
				DatePublished: jsonStr(r, "age"),
			})
		}
	}

	resp := &SearchResponse{
		Provider: "brave",
		Query:    request.Query,
		Results:  results,
	}

	if searchType == SearchTypeWeb {
		if q := jsonObj(data, "query"); q != nil {
			for _, rs := range jsonArray(q, "related_searches") {
				query := jsonStr(rs, "query")
				if query == "" {
					if s, ok := rs["_str"]; ok {
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
	}

	return resp, nil
}

func (a *BraveAdapter) makeRequest(ctx context.Context, endpoint string, params url.Values) (map[string]interface{}, error) {
	u := "https://api.search.brave.com/res/v1" + endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "brave"})
		}
		return nil, fmt.Errorf("brave: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "message")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "brave", "raw": result})
	}

	return result, nil
}
