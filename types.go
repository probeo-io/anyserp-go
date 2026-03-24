// Package anyserp provides a unified SERP API router supporting 11 search providers.
package anyserp

import "context"

// Version is the current library version.
const Version = "0.1.0"

// SearchType specifies the kind of search to perform.
type SearchType string

const (
	SearchTypeWeb    SearchType = "web"
	SearchTypeImages SearchType = "images"
	SearchTypeNews   SearchType = "news"
	SearchTypeVideos SearchType = "videos"
)

// DateRange specifies a time-based filter for search results.
type DateRange string

const (
	DateRangeDay   DateRange = "day"
	DateRangeWeek  DateRange = "week"
	DateRangeMonth DateRange = "month"
	DateRangeYear  DateRange = "year"
)

// SearchRequest holds parameters for a search query.
type SearchRequest struct {
	Query             string     `json:"query"`
	Num               int        `json:"num,omitempty"`
	Page              int        `json:"page,omitempty"`
	Country           string     `json:"country,omitempty"`
	Language          string     `json:"language,omitempty"`
	Safe              bool       `json:"safe,omitempty"`
	Type              SearchType `json:"type,omitempty"`
	DateRange         DateRange  `json:"dateRange,omitempty"`
	IncludeAiOverview bool       `json:"includeAiOverview,omitempty"`
}

// SearchResponse holds the results of a search query.
type SearchResponse struct {
	Provider        string          `json:"provider"`
	Query           string          `json:"query"`
	Results         []SearchResult  `json:"results"`
	TotalResults    int             `json:"totalResults,omitempty"`
	SearchTime      float64         `json:"searchTime,omitempty"`
	RelatedSearches []string        `json:"relatedSearches,omitempty"`
	PeopleAlsoAsk   []PeopleAlsoAsk `json:"peopleAlsoAsk,omitempty"`
	KnowledgePanel  *KnowledgePanel `json:"knowledgePanel,omitempty"`
	AnswerBox       *AnswerBox      `json:"answerBox,omitempty"`
	AiOverview      *AiOverview     `json:"aiOverview,omitempty"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	Position      int    `json:"position"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	Description   string `json:"description"`
	Domain        string `json:"domain,omitempty"`
	DatePublished string `json:"datePublished,omitempty"`
	Thumbnail     string `json:"thumbnail,omitempty"`
	// Image-specific
	ImageURL    string `json:"imageUrl,omitempty"`
	ImageWidth  int    `json:"imageWidth,omitempty"`
	ImageHeight int    `json:"imageHeight,omitempty"`
	// News-specific
	Source string `json:"source,omitempty"`
	// Video-specific
	Duration string `json:"duration,omitempty"`
	Channel  string `json:"channel,omitempty"`
}

// PeopleAlsoAsk represents a "People Also Ask" item.
type PeopleAlsoAsk struct {
	Question string `json:"question"`
	Snippet  string `json:"snippet,omitempty"`
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"`
}

// KnowledgePanel represents a knowledge graph panel.
type KnowledgePanel struct {
	Title       string            `json:"title"`
	Type        string            `json:"type,omitempty"`
	Description string            `json:"description,omitempty"`
	Source      string            `json:"source,omitempty"`
	SourceURL   string            `json:"sourceUrl,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	ImageURL    string            `json:"imageUrl,omitempty"`
}

// AnswerBox represents a featured answer box.
type AnswerBox struct {
	Snippet string `json:"snippet"`
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
}

// AiOverview represents an AI-generated overview from search results.
type AiOverview struct {
	Markdown   string                `json:"markdown,omitempty"`
	TextBlocks []AiOverviewTextBlock `json:"textBlocks"`
	References []AiOverviewReference `json:"references"`
	PageToken  string                `json:"pageToken,omitempty"`
}

// AiOverviewTextBlock represents a structured block within an AI overview.
type AiOverviewTextBlock struct {
	Type             string                `json:"type"`
	Answer           string                `json:"answer,omitempty"`
	AnswerHighlight  string                `json:"answerHighlight,omitempty"`
	Items            []AiOverviewTextBlock `json:"items,omitempty"`
	Table            *AiOverviewTable      `json:"table,omitempty"`
	Language         string                `json:"language,omitempty"`
	Code             string                `json:"code,omitempty"`
	Video            *AiOverviewVideo      `json:"video,omitempty"`
	ReferenceIndexes []int                 `json:"referenceIndexes,omitempty"`
	Link             string                `json:"link,omitempty"`
	RelatedSearches  []RelatedSearch       `json:"relatedSearches,omitempty"`
}

// AiOverviewTable holds tabular data within an AI overview block.
type AiOverviewTable struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// AiOverviewVideo holds video metadata within an AI overview block.
type AiOverviewVideo struct {
	Title    string `json:"title,omitempty"`
	Link     string `json:"link,omitempty"`
	Duration string `json:"duration,omitempty"`
	Source   string `json:"source,omitempty"`
	Channel  string `json:"channel,omitempty"`
}

// RelatedSearch represents a related search suggestion.
type RelatedSearch struct {
	Query string `json:"query"`
	Link  string `json:"link,omitempty"`
}

// AiOverviewReference represents a source cited in an AI overview.
type AiOverviewReference struct {
	Index     int    `json:"index"`
	Title     string `json:"title,omitempty"`
	URL       string `json:"url,omitempty"`
	Snippet   string `json:"snippet,omitempty"`
	Date      string `json:"date,omitempty"`
	Source    string `json:"source,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// SearchAdapter defines the interface that all search providers implement.
type SearchAdapter interface {
	Name() string
	Search(ctx context.Context, request SearchRequest) (*SearchResponse, error)
	SupportsType(searchType SearchType) bool
}

// ProviderConfig holds API credentials for a search provider.
type ProviderConfig struct {
	APIKey   string
	EngineID string // Google CSE only
}

// DataForSeoConfig holds credentials for the DataForSEO provider.
type DataForSeoConfig struct {
	Login    string
	Password string
}

// SearchDefaults holds default values applied to all search requests.
type SearchDefaults struct {
	Num      int
	Country  string
	Language string
	Safe     bool
}

// Config holds configuration for all search providers.
type Config struct {
	Serper      *ProviderConfig
	SerpAPI     *ProviderConfig
	Google      *ProviderConfig
	Bing        *ProviderConfig
	Brave       *ProviderConfig
	DataForSEO  *DataForSeoConfig
	SearchAPI   *ProviderConfig
	ValueSERP   *ProviderConfig
	ScrapingDog *ProviderConfig
	BrightData  *ProviderConfig
	SearchCans  *ProviderConfig
	Defaults    *SearchDefaults
	Aliases     map[string]string
}
