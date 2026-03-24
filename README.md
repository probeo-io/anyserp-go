# anyserp-go

Unified SERP API router for Go. Route search requests across Google, Bing, Brave, and more with a single API. Self-hosted, zero fees.

## Install

```bash
go get github.com/probeo-io/anyserp-go
```

## Quick Start

Set your API keys as environment variables:

```bash
export SERPER_API_KEY=...
export BRAVE_API_KEY=...
```

```go
package main

import (
    "context"
    "fmt"

    serp "github.com/probeo-io/anyserp-go"
)

func main() {
    client := serp.New(nil) // reads API keys from env vars

    // Search with the first available provider
    results, err := client.Search(context.Background(), serp.SearchRequest{
        Query: "best go frameworks",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(results.Results[0].Title, results.Results[0].URL)
}
```

## Supported Providers

| Provider | Env Var | Web | Images | News | Videos |
|----------|---------|-----|--------|------|--------|
| Serper | `SERPER_API_KEY` | Yes | Yes | Yes | Yes |
| SerpAPI | `SERPAPI_API_KEY` | Yes | Yes | Yes | Yes |
| Google CSE | `GOOGLE_CSE_API_KEY` + `GOOGLE_CSE_ENGINE_ID` | Yes | Yes | No | No |
| Bing | `BING_API_KEY` | Yes | Yes | Yes | Yes |
| Brave | `BRAVE_API_KEY` | Yes | Yes | Yes | Yes |
| DataForSEO | `DATAFORSEO_LOGIN` + `DATAFORSEO_PASSWORD` | Yes | No | Yes | No |
| SearchAPI | `SEARCHAPI_API_KEY` | Yes | Yes | Yes | Yes |
| ValueSERP | `VALUESERP_API_KEY` | Yes | Yes | Yes | Yes |
| ScrapingDog | `SCRAPINGDOG_API_KEY` | Yes | Yes | Yes | No |
| Bright Data | `BRIGHTDATA_API_KEY` | Yes | Yes | Yes | Yes |
| SearchCans | `SEARCHCANS_API_KEY` | Yes | No | Yes | No |

## Provider Routing

Specify a provider with `provider/query` format:

```go
// Use a specific provider
results, _ := client.Search(ctx, serp.SearchRequest{Query: "serper/go frameworks"})

// Or just search with the first available
results, _ := client.Search(ctx, serp.SearchRequest{Query: "go frameworks"})
```

## Search Options

```go
results, err := client.Search(ctx, serp.SearchRequest{
    Query:     "go frameworks",
    Num:       20,                    // number of results
    Page:      2,                     // page number
    Country:   "us",                  // country code
    Language:  "en",                  // language code
    Safe:      true,                  // safe search
    Type:      serp.SearchTypeWeb,    // web, images, news, videos
    DateRange: serp.DateRangeMonth,   // day, week, month, year
})
```

## Fallback Routing

Try multiple providers in order. If one fails, the next is attempted:

```go
results, err := client.SearchWithFallback(ctx, serp.SearchRequest{
    Query: "go frameworks",
}, []string{"serper", "brave", "bing"})
```

## Unified Response Format

All providers return the same `SearchResponse` struct:

```go
type SearchResponse struct {
    Provider       string
    Query          string
    Results        []SearchResult
    TotalResults   int
    SearchTime     float64 // ms
    RelatedSearches []string
    PeopleAlsoAsk  []PeopleAlsoAsk
    KnowledgePanel *KnowledgePanel
    AnswerBox      *AnswerBox
    AiOverview     *AiOverview
}

type SearchResult struct {
    Position      int
    Title         string
    URL           string
    Description   string
    Domain        string
    DatePublished string
    // Image fields
    ImageURL      string
    ImageWidth    int
    ImageHeight   int
    // News fields
    Source        string
    // Video fields
    Duration      string
    Channel       string
}
```

## Configuration

### Programmatic

```go
client := serp.New(&serp.Config{
    Serper:      &serp.ProviderConfig{APIKey: "..."},
    Brave:       &serp.ProviderConfig{APIKey: "..."},
    Google:      &serp.ProviderConfig{APIKey: "...", EngineID: "..."},
    DataForSEO:  &serp.DataForSeoConfig{Login: "...", Password: "..."},
    SearchAPI:   &serp.ProviderConfig{APIKey: "..."},
    ValueSERP:   &serp.ProviderConfig{APIKey: "..."},
    ScrapingDog: &serp.ProviderConfig{APIKey: "..."},
    BrightData:  &serp.ProviderConfig{APIKey: "..."},
    SearchCans:  &serp.ProviderConfig{APIKey: "..."},
    Defaults: &serp.SearchDefaults{
        Num:      10,
        Country:  "us",
        Language: "en",
        Safe:     true,
    },
    Aliases: map[string]string{
        "fast":    "serper",
        "default": "brave",
    },
})
```

### Environment Variables

```bash
export SERPER_API_KEY=...
export SERPAPI_API_KEY=...
export GOOGLE_CSE_API_KEY=...
export GOOGLE_CSE_ENGINE_ID=...
export BING_API_KEY=...
export BRAVE_API_KEY=...
export DATAFORSEO_LOGIN=...
export DATAFORSEO_PASSWORD=...
export SEARCHAPI_API_KEY=...
export VALUESERP_API_KEY=...
export SCRAPINGDOG_API_KEY=...
export BRIGHTDATA_API_KEY=...
export SEARCHCANS_API_KEY=...
```

## People Also Ask

Available from 8 providers (Serper, SerpAPI, SearchAPI, ValueSERP, DataForSEO, ScrapingDog, Bright Data, SearchCans):

```go
results, _ := client.Search(ctx, serp.SearchRequest{Query: "how to start an LLC"})
for _, paa := range results.PeopleAlsoAsk {
    fmt.Println(paa.Question, paa.Snippet)
}
```

## AI Overview

Fetch Google's AI-generated overview content (requires SearchAPI):

```go
results, _ := client.Search(ctx, serp.SearchRequest{
    Query:             "how to start an LLC",
    IncludeAiOverview: true,
})

if results.AiOverview != nil {
    fmt.Println(results.AiOverview.Markdown)
    for _, ref := range results.AiOverview.References {
        fmt.Printf("  [%d] %s - %s\n", ref.Index, ref.Title, ref.URL)
    }
}
```

## Also Available

- **Node.js**: [`@probeo/anyserp`](https://github.com/probeo-io/anyserp) on npm
- **Python**: [`anyserp`](https://github.com/probeo-io/anyserp-py) on PyPI

## License

MIT
