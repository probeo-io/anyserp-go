package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	as "github.com/probeo-io/anyserp-go"
)

func main() {
	args := os.Args[1:]
	demos := map[string]func(context.Context){
		"search":   demoSearch,
		"provider": demoProvider,
		"images":   demoImages,
		"news":     demoNews,
	}

	if len(args) == 0 {
		for _, fn := range demos {
			fn(context.Background())
		}
		return
	}

	for _, name := range args {
		fn, ok := demos[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown demo: %s\nAvailable: %s\n", name, strings.Join(keys(demos), ", "))
			os.Exit(1)
		}
		fn(context.Background())
	}
}

func demoSearch(ctx context.Context) {
	fmt.Println("\n=== Basic Search ===")
	client := as.New(nil)

	results, err := client.Search(ctx, &as.SearchRequest{
		Query: "best golang frameworks 2026",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Provider: %s\n", results.Provider)
	fmt.Printf("Results: %d\n\n", len(results.Results))
	limit := 3
	if len(results.Results) < limit {
		limit = len(results.Results)
	}
	for _, r := range results.Results[:limit] {
		fmt.Printf("  %s\n  %s\n\n", r.Title, r.URL)
	}
}

func demoProvider(ctx context.Context) {
	fmt.Println("\n=== Provider-Specific Search ===")
	client := as.New(nil)

	results, err := client.Search(ctx, &as.SearchRequest{
		Query:    "go concurrency patterns",
		Provider: "serper",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Provider: %s\n", results.Provider)
	limit := 3
	if len(results.Results) < limit {
		limit = len(results.Results)
	}
	for _, r := range results.Results[:limit] {
		fmt.Printf("  %s\n", r.Title)
	}
	fmt.Println()
}

func demoImages(ctx context.Context) {
	fmt.Println("\n=== Image Search ===")
	client := as.New(nil)

	results, err := client.SearchImages(ctx, &as.SearchRequest{
		Query: "aurora borealis",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Images found: %d\n", len(results.Results))
	limit := 3
	if len(results.Results) < limit {
		limit = len(results.Results)
	}
	for _, r := range results.Results[:limit] {
		fmt.Printf("  %s - %s\n", r.Title, r.URL)
	}
	fmt.Println()
}

func demoNews(ctx context.Context) {
	fmt.Println("\n=== News Search ===")
	client := as.New(nil)

	results, err := client.SearchNews(ctx, &as.SearchRequest{
		Query: "artificial intelligence",
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	limit := 3
	if len(results.Results) < limit {
		limit = len(results.Results)
	}
	for _, r := range results.Results[:limit] {
		fmt.Printf("  %s\n  %s\n\n", r.Title, r.URL)
	}
}

func keys(m map[string]func(context.Context)) []string {
	k := make([]string, 0, len(m))
	for key := range m {
		k = append(k, key)
	}
	return k
}
