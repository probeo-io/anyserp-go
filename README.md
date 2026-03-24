# anyserp-go

Unified SERP API router for Go. 11 providers — one interface. Self-hosted, zero fees.

## Install

```bash
go get github.com/probeo-io/anyserp-go
```

## Quick Start

```bash
export SERPER_API_KEY=...
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

    results, err := client.Search(context.Background(), serp.SearchRequest{
        Query: "best go frameworks",
    })
    if err != nil {
        panic(err)
    }
    for _, r := range results.Results {
        fmt.Println(r.Title, r.URL)
    }
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

```go
// Use a specific provider
results, _ := client.Search(ctx, serp.SearchRequest{Query: "serper/go frameworks"})

// Or search with the first available
results, _ := client.Search(ctx, serp.SearchRequest{Query: "go frameworks"})
```

## Search Options

```go
results, err := client.Search(ctx, serp.SearchRequest{
    Query:     "go frameworks",
    Num:       20,
    Page:      2,
    Country:   "us",
    Language:  "en",
    Safe:      true,
    Type:      serp.SearchTypeWeb,
    DateRange: serp.DateRangeMonth,
})
```

## Fallback Routing

```go
results, err := client.SearchWithFallback(ctx, serp.SearchRequest{
    Query: "go frameworks",
}, []string{"serper", "brave", "bing"})
```

## People Also Ask

Available from 8 providers:

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

## Configuration

```go
client := serp.New(&serp.Config{
    Serper:  &serp.ProviderConfig{APIKey: "..."},
    Brave:   &serp.ProviderConfig{APIKey: "..."},
    Defaults: &serp.SearchDefaults{Num: 10, Country: "us"},
    Aliases: map[string]string{"fast": "serper"},
})
```

## Also Available

- **Node.js**: [`@probeo/anyserp`](https://github.com/probeo-io/anyserp) on npm
- **Python**: [`anyserp`](https://github.com/probeo-io/anyserp-py) on PyPI

## License

MIT
