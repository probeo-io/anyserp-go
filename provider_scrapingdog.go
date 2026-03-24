package anyserp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var scrapingDogEndpointMap = map[SearchType]string{
	SearchTypeWeb:    "/google",
	SearchTypeImages: "/google_images",
	SearchTypeNews:   "/google_news",
	SearchTypeVideos: "/google",
}

// ScrapingDogAdapter implements SearchAdapter for the ScrapingDog API.
type ScrapingDogAdapter struct {
	apiKey string
	client *http.Client
}

// NewScrapingDogAdapter creates a new ScrapingDogAdapter.
func NewScrapingDogAdapter(apiKey string, client *http.Client) *ScrapingDogAdapter {
	return &ScrapingDogAdapter{apiKey: apiKey, client: client}
}

func (a *ScrapingDogAdapter) Name() string { return "scrapingdog" }

func (a *ScrapingDogAdapter) SupportsType(t SearchType) bool {
	return t == SearchTypeWeb || t == SearchTypeImages || t == SearchTypeNews
}

func (a *ScrapingDogAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}
	endpoint := scrapingDogEndpointMap[searchType]

	params := url.Values{}
	params.Set("api_key", a.apiKey)
	params.Set("query", request.Query)

	if request.Num > 0 {
		params.Set("results", strconv.Itoa(request.Num))
	}
	if request.Page > 1 {
		params.Set("page", strconv.Itoa(request.Page-1)) // 0-indexed
	}
	if request.Country != "" {
		params.Set("country", request.Country)
	}
	if request.Language != "" {
		params.Set("language", request.Language)
	}

	data, err := a.makeRequest(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	switch searchType {
	case SearchTypeWeb:
		arr := jsonArray(data, "organic_results")
		if len(arr) == 0 {
			arr = jsonArray(data, "organic_data")
		}
		for i, r := range arr {
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
	case SearchTypeImages:
		arr := jsonArray(data, "image_results")
		for i, r := range arr {
			u := jsonStr(r, "link")
			if u == "" {
				u = jsonStr(r, "url")
			}
			imageURL := jsonStr(r, "original")
			if imageURL == "" {
				imageURL = jsonStr(r, "image")
			}
			results = append(results, SearchResult{
				Position:    i + 1,
				Title:       jsonStr(r, "title"),
				URL:         u,
				Description: jsonStr(r, "title"),
				ImageURL:    imageURL,
				ImageWidth:  jsonInt(r, "original_width"),
				ImageHeight: jsonInt(r, "original_height"),
				Thumbnail:   jsonStr(r, "thumbnail"),
				Source:      jsonStr(r, "source"),
			})
		}
	case SearchTypeNews:
		arr := jsonArray(data, "news_results")
		for i, r := range arr {
			u := jsonStr(r, "link")
			if u == "" {
				u = jsonStr(r, "url")
			}
			desc := jsonStr(r, "snippet")
			if desc == "" {
				desc = jsonStr(r, "description")
			}
			thumbnail := jsonStr(r, "thumbnail")
			if thumbnail == "" {
				thumbnail = jsonStr(r, "image")
			}
			results = append(results, SearchResult{
				Position:      i + 1,
				Title:         jsonStr(r, "title"),
				URL:           u,
				Description:   desc,
				Source:        jsonStr(r, "source"),
				DatePublished: jsonStr(r, "date"),
				Thumbnail:     thumbnail,
			})
		}
	}

	resp := &SearchResponse{
		Provider: "scrapingdog",
		Query:    request.Query,
		Results:  results,
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

	return resp, nil
}

func (a *ScrapingDogAdapter) makeRequest(ctx context.Context, endpoint string, params url.Values) (map[string]interface{}, error) {
	u := "https://api.scrapingdog.com" + endpoint + "?" + params.Encode()

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
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "scrapingdog"})
		}
		return nil, fmt.Errorf("scrapingdog: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "error")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "scrapingdog", "raw": result})
	}

	return result, nil
}
