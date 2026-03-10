package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"auto-search/internal/config"
	"auto-search/internal/httpclient"
	"auto-search/internal/query"
)

const googleNewsBaseURL = "https://news.google.com/rss/search"

type Item struct {
	QueryID        int64
	QueryName      string
	QueryText      string
	QueryLang      string
	QueryRegion    string
	Title          string
	Link           string
	Description    string
	PublishedAtRaw string
	SourceSite     string
	GUID           string
}

type GoogleNewsClient struct {
	httpClient *http.Client
	userAgent  string
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	GUID        string    `xml:"guid"`
	Source      rssSource `xml:"source"`
}

type rssSource struct {
	Name string `xml:",chardata"`
	URL  string `xml:"url,attr"`
}

func NewGoogleNewsClient(cfg config.HTTPConfig) *GoogleNewsClient {
	return &GoogleNewsClient{
		httpClient: httpclient.New(cfg),
		userAgent:  cfg.UserAgent,
	}
}

func BuildFeedURL(q query.FeedQuery) string {
	values := url.Values{}
	values.Set("q", q.QueryText)
	values.Set("hl", normalizeLang(q.Lang))
	values.Set("gl", normalizeRegion(q.Region))
	values.Set("ceid", fmt.Sprintf("%s:%s", normalizeRegion(q.Region), normalizeLang(q.Lang)))
	return googleNewsBaseURL + "?" + values.Encode()
}

func (c *GoogleNewsClient) Fetch(ctx context.Context, q query.FeedQuery) ([]Item, error) {
	return c.fetchFromURL(ctx, q, BuildFeedURL(q))
}

func (c *GoogleNewsClient) fetchFromURL(ctx context.Context, q query.FeedQuery, feedURL string) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 RSS 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml;q=0.9")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 RSS 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RSS 返回状态异常: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 RSS 响应失败: %w", err)
	}

	var doc rssDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("解析 RSS XML 失败: %w", err)
	}

	items := make([]Item, 0, len(doc.Channel.Items))
	for _, rssItem := range doc.Channel.Items {
		items = append(items, Item{
			QueryID:        q.ID,
			QueryName:      q.Name,
			QueryText:      q.QueryText,
			QueryLang:      q.Lang,
			QueryRegion:    q.Region,
			Title:          strings.TrimSpace(rssItem.Title),
			Link:           strings.TrimSpace(rssItem.Link),
			Description:    strings.TrimSpace(rssItem.Description),
			PublishedAtRaw: strings.TrimSpace(rssItem.PubDate),
			SourceSite:     strings.TrimSpace(rssItem.Source.Name),
			GUID:           strings.TrimSpace(rssItem.GUID),
		})
	}

	return items, nil
}

func normalizeLang(lang string) string {
	if strings.TrimSpace(lang) == "" {
		return "en"
	}
	return strings.ToLower(lang)
}

func normalizeRegion(region string) string {
	if strings.TrimSpace(region) == "" {
		return "US"
	}
	return strings.ToUpper(region)
}
