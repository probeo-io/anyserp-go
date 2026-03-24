package anyserp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

var dataForSeoCountryMap = map[string]int{
	"us": 2840, "gb": 2826, "ca": 2124, "au": 2036, "de": 2276, "fr": 2250,
	"es": 2724, "it": 2380, "br": 2076, "in": 2356, "jp": 2392, "kr": 2410,
	"mx": 2484, "nl": 2528, "se": 2752, "no": 2578, "dk": 2208, "fi": 2246,
	"pl": 2616, "ru": 2643, "za": 2710, "ar": 2032, "cl": 2152, "co": 2170,
	"pt": 2620, "be": 2056, "at": 2040, "ch": 2756, "ie": 2372, "nz": 2554,
	"sg": 2702, "hk": 2344, "tw": 2158, "ph": 2608, "th": 2764, "my": 2458,
	"id": 2360, "vn": 2704, "tr": 2792, "il": 2376, "ae": 2784, "sa": 2682,
	"eg": 2818, "ng": 2566, "ke": 2404,
}

var dataForSeoSETypeMap = map[SearchType]string{
	SearchTypeWeb:    "organic",
	SearchTypeImages: "organic",
	SearchTypeNews:   "news",
	SearchTypeVideos: "organic",
}

// DataForSeoAdapter implements SearchAdapter for the DataForSEO API.
type DataForSeoAdapter struct {
	authHeader string
	client     *http.Client
}

// NewDataForSeoAdapter creates a new DataForSeoAdapter.
func NewDataForSeoAdapter(login, password string, client *http.Client) *DataForSeoAdapter {
	auth := base64.StdEncoding.EncodeToString([]byte(login + ":" + password))
	return &DataForSeoAdapter{authHeader: "Basic " + auth, client: client}
}

func (a *DataForSeoAdapter) Name() string { return "dataforseo" }

func (a *DataForSeoAdapter) SupportsType(t SearchType) bool {
	return t == SearchTypeWeb || t == SearchTypeNews
}

func (a *DataForSeoAdapter) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	searchType := request.Type
	if searchType == "" {
		searchType = SearchTypeWeb
	}

	seType := dataForSeoSETypeMap[searchType]
	path := fmt.Sprintf("/serp/google/%s/live/advanced", seType)

	task := map[string]interface{}{
		"keyword": request.Query,
		"depth":   request.Num,
	}
	if task["depth"] == 0 {
		task["depth"] = 10
	}

	if request.Country != "" {
		if code, ok := dataForSeoCountryMap[request.Country]; ok {
			task["location_code"] = code
		}
	}
	if request.Language != "" {
		task["language_code"] = request.Language
	}
	if request.Page > 1 {
		num := request.Num
		if num == 0 {
			num = 10
		}
		task["depth"] = num * request.Page
	}

	taskResult, err := a.makeRequest(ctx, path, []map[string]interface{}{task})
	if err != nil {
		return nil, err
	}

	resultArr := jsonArray(taskResult, "result")
	var resultData map[string]interface{}
	if len(resultArr) > 0 {
		resultData = resultArr[0]
	}

	var results []SearchResult
	var knowledgePanel *KnowledgePanel
	var answerBox *AnswerBox
	var paaQuestions []PeopleAlsoAsk

	if resultData != nil {
		items := jsonArray(resultData, "items")
		position := 0
		for _, item := range items {
			itemType := jsonStr(item, "type")
			switch itemType {
			case "organic":
				position++
				results = append(results, SearchResult{
					Position:      position,
					Title:         jsonStr(item, "title"),
					URL:           jsonStr(item, "url"),
					Description:   jsonStr(item, "description"),
					Domain:        jsonStr(item, "domain"),
					DatePublished: jsonStr(item, "timestamp"),
				})
			case "knowledge_graph":
				knowledgePanel = &KnowledgePanel{
					Title:       jsonStr(item, "title"),
					Type:        jsonStr(item, "sub_title"),
					Description: jsonStr(item, "description"),
					ImageURL:    jsonStr(item, "image_url"),
				}
			case "featured_snippet":
				snippet := jsonStr(item, "description")
				if snippet == "" {
					snippet = jsonStr(item, "title")
				}
				answerBox = &AnswerBox{
					Snippet: snippet,
					Title:   jsonStr(item, "title"),
					URL:     jsonStr(item, "url"),
				}
			case "people_also_ask":
				for _, q := range jsonArray(item, "items") {
					paaQuestions = append(paaQuestions, PeopleAlsoAsk{
						Question: jsonStr(q, "title"),
						Snippet:  jsonStr(q, "description"),
						URL:      jsonStr(q, "url"),
					})
				}
			case "news_search":
				if searchType == SearchTypeNews {
					position++
					desc := jsonStr(item, "snippet")
					if desc == "" {
						desc = jsonStr(item, "description")
					}
					dp := jsonStr(item, "timestamp")
					if dp == "" {
						dp = jsonStr(item, "datetime")
					}
					results = append(results, SearchResult{
						Position:      position,
						Title:         jsonStr(item, "title"),
						URL:           jsonStr(item, "url"),
						Description:   desc,
						Source:        jsonStr(item, "source"),
						DatePublished: dp,
						Thumbnail:     jsonStr(item, "image_url"),
					})
				}
			}
		}
	}

	resp := &SearchResponse{
		Provider:       "dataforseo",
		Query:          request.Query,
		Results:        results,
		KnowledgePanel: knowledgePanel,
		AnswerBox:      answerBox,
	}

	if resultData != nil {
		resp.TotalResults = jsonInt(resultData, "se_results_count")
	}
	if len(paaQuestions) > 0 {
		resp.PeopleAlsoAsk = paaQuestions
	}

	return resp, nil
}

func (a *DataForSeoAdapter) makeRequest(ctx context.Context, path string, tasks []map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(tasks)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.dataforseo.com/v3"+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", a.authHeader)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, NewAnySerpError(resp.StatusCode, resp.Status, map[string]interface{}{"provider_name": "dataforseo"})
		}
		return nil, fmt.Errorf("dataforseo: failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := jsonStr(result, "status_message")
		if msg == "" {
			msg = resp.Status
		}
		return nil, NewAnySerpError(resp.StatusCode, msg, map[string]interface{}{"provider_name": "dataforseo", "raw": result})
	}

	statusCode := jsonInt(result, "status_code")
	if statusCode >= 40000 {
		code := 400
		if statusCode >= 50000 {
			code = 502
		}
		msg := jsonStr(result, "status_message")
		if msg == "" {
			msg = "DataForSEO error"
		}
		return nil, NewAnySerpError(code, msg, map[string]interface{}{"provider_name": "dataforseo", "raw": result})
	}

	tasksArr := jsonArray(result, "tasks")
	if len(tasksArr) == 0 {
		return nil, NewAnySerpError(502, "No task in DataForSEO response", map[string]interface{}{"provider_name": "dataforseo", "raw": result})
	}

	task := tasksArr[0]
	taskStatusCode := jsonInt(task, "status_code")
	if taskStatusCode >= 40000 {
		code := 400
		if taskStatusCode >= 50000 {
			code = 502
		}
		msg := jsonStr(task, "status_message")
		if msg == "" {
			msg = "DataForSEO task error"
		}
		return nil, NewAnySerpError(code, msg, map[string]interface{}{"provider_name": "dataforseo", "raw": task})
	}

	return task, nil
}
