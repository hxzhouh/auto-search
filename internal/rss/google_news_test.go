package rss

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"auto-search/internal/config"
	"auto-search/internal/query"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestBuildFeedURL(t *testing.T) {
	t.Parallel()

	feedURL := BuildFeedURL(query.FeedQuery{
		QueryText: `OpenAI when:1d`,
		Lang:      "en",
		Region:    "us",
	})

	if !strings.Contains(feedURL, "q=OpenAI+when%3A1d") {
		t.Fatalf("query 参数编码不正确: %s", feedURL)
	}
	if !strings.Contains(feedURL, "hl=en") || !strings.Contains(feedURL, "gl=US") {
		t.Fatalf("地区和语言参数不正确: %s", feedURL)
	}
}

func TestGoogleNewsClientFetch(t *testing.T) {
	t.Parallel()

	client := &GoogleNewsClient{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>OpenAI 发布新模型</title>
      <link>https://news.google.com/articles/abc</link>
      <description>一段摘要</description>
      <pubDate>Tue, 10 Mar 2026 10:00:00 +0800</pubDate>
      <guid>guid-1</guid>
      <source url="https://example.com">Example</source>
    </item>
  </channel>
</rss>`

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
		userAgent: "auto-search-test",
	}

	items, err := client.fetchFromURL(context.Background(), query.FeedQuery{
		ID:        1,
		Name:      "openai",
		QueryText: "OpenAI when:1d",
		Lang:      "en",
		Region:    "US",
	}, "https://example.com/feed")
	if err != nil {
		t.Fatalf("抓取 RSS 失败: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("期望 1 条 RSS，实际为 %d", len(items))
	}
	if items[0].SourceSite != "Example" {
		t.Fatalf("期望来源站点为 Example，实际为 %s", items[0].SourceSite)
	}
}

func TestNewGoogleNewsClient(t *testing.T) {
	t.Parallel()

	client := NewGoogleNewsClient(config.HTTPConfig{
		TimeoutSeconds: 10,
		UserAgent:      "auto-search-test",
		MaxRedirects:   5,
	})
	if client == nil || client.httpClient == nil {
		t.Fatal("客户端初始化失败")
	}
}
