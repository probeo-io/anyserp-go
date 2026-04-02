# Contributing to anyserp-go

Thanks for your interest in contributing! Here's how to get started.

## Setup

```bash
git clone https://github.com/probeo-io/anyserp-go.git
cd anyserp-go
go mod download
```

## Development

```bash
# Run tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -run TestSearchSerper ./...
```

## Project Structure

```
client.go              # AnySerp client, Search() entry point
types.go               # All shared types (SearchRequest, SearchResponse, etc.)
errors.go              # Error types
json_helpers.go        # JSON parsing utilities
provider_serper.go     # Serper adapter
provider_serpapi.go     # SerpAPI adapter
provider_google.go     # Google CSE adapter
provider_bing.go       # Bing adapter
provider_brave.go      # Brave adapter
provider_dataforseo.go # DataForSEO adapter
provider_searchapi.go  # SearchAPI adapter
provider_valueserp.go  # ValueSERP adapter
provider_scrapingdog.go # ScrapingDog adapter
provider_brightdata.go # Bright Data adapter
provider_searchcans.go # SearchCans adapter
*_test.go              # Test files
```

## Making Changes

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Add or update tests as needed
4. Run `go test ./...` to make sure everything passes
5. Write a clear commit message describing what and why
6. Open a pull request

## Pull Requests

- Keep PRs focused. One feature or fix per PR
- Include tests for new functionality
- Update the README if you're adding user-facing features
- Make sure CI passes before requesting review

## Adding a Provider

1. Create `provider_yourprovider.go` implementing the provider interface
2. Register it in `client.go`
3. Add tests in `provider_test.go` or a new test file
4. Update the README with the new provider

## Reporting Issues

Use [GitHub Issues](https://github.com/probeo-io/anyserp-go/issues). Include:

- What you expected to happen
- What actually happened
- Steps to reproduce
- Go version and OS

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- No external runtime dependencies (stdlib only)
- Keep things simple. No premature abstractions
