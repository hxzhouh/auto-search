package extractor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"auto-search/internal/config"
	"auto-search/internal/httpclient"
)

type DefuddleClient struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

type Article struct {
	Title       string
	Author      string
	Description string
	PublishedAt *time.Time
	Markdown    string
	RawMarkdown string
}

func NewDefuddleClient(cfg *config.Config) *DefuddleClient {
	return &DefuddleClient{
		httpClient: httpclient.New(cfg.HTTP),
		baseURL:    strings.TrimRight(cfg.Defuddle.BaseURL, "/"),
		userAgent:  cfg.HTTP.UserAgent,
	}
}

func (c *DefuddleClient) Extract(ctx context.Context, targetURL string) (Article, error) {
	requestURL, err := buildDefuddleURL(c.baseURL, targetURL)
	if err != nil {
		return Article{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Article{}, fmt.Errorf("创建 defuddle 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/markdown, text/plain;q=0.9, */*;q=0.8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Article{}, fmt.Errorf("请求 defuddle 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Article{}, fmt.Errorf("defuddle 返回状态异常: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Article{}, fmt.Errorf("读取 defuddle 响应失败: %w", err)
	}

	return parseMarkdownDocument(string(body))
}

func buildDefuddleURL(baseURL, targetURL string) (string, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("解析目标 URL 失败: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("目标 URL 缺少 host: %s", targetURL)
	}

	trimmed := parsed.Host
	if parsed.Path != "" {
		trimmed += parsed.EscapedPath()
	}

	requestURL := baseURL + "/" + strings.TrimLeft(trimmed, "/")
	if parsed.RawQuery != "" {
		requestURL += "?" + parsed.RawQuery
	}
	return requestURL, nil
}

func parseMarkdownDocument(raw string) (Article, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Article{}, fmt.Errorf("defuddle 返回内容为空")
	}

	if !strings.HasPrefix(raw, "---\n") {
		return Article{
			Markdown:    raw,
			RawMarkdown: raw,
		}, nil
	}

	sections := strings.SplitN(raw, "\n---\n", 2)
	if len(sections) != 2 {
		return Article{
			Markdown:    raw,
			RawMarkdown: raw,
		}, nil
	}

	meta := parseFrontmatter(strings.TrimPrefix(sections[0], "---\n"))
	body := strings.TrimSpace(sections[1])

	article := Article{
		Title:       meta["title"],
		Author:      meta["author"],
		Description: meta["description"],
		Markdown:    body,
		RawMarkdown: raw,
	}

	if published := strings.TrimSpace(meta["published"]); published != "" {
		if parsed := parsePublishedAt(published); parsed != nil {
			article.PublishedAt = parsed
		}
	}

	return article, nil
}

func parseFrontmatter(raw string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		result[key] = value
	}

	return result
}

func parsePublishedAt(raw string) *time.Time {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
		"2006-01-02 15:04:05",
		time.RFC1123Z,
		time.RFC1123,
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return &parsed
		}
	}

	return nil
}
