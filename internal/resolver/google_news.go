package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"auto-search/internal/config"
	"auto-search/internal/httpclient"
)

type Result struct {
	FinalURL     string
	CanonicalURL string
}

type GoogleNewsResolver struct {
	httpClient *http.Client
	userAgent  string
}

var (
	signaturePattern = regexp.MustCompile(`data-n-a-sg="([^"]+)"`)
	timestampPattern = regexp.MustCompile(`data-n-a-ts="([^"]+)"`)
)

func NewGoogleNewsResolver(cfg config.HTTPConfig) *GoogleNewsResolver {
	return &GoogleNewsResolver{
		httpClient: httpclient.New(cfg),
		userAgent:  cfg.UserAgent,
	}
}

func (r *GoogleNewsResolver) Resolve(ctx context.Context, rawURL string) (Result, error) {
	if isGoogleNewsArticleURL(rawURL) {
		return r.resolveGoogleNewsArticle(ctx, rawURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Result{}, fmt.Errorf("创建链接解析请求失败: %w", err)
	}
	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("请求跳转链接失败: %w", err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))

	finalURL := resp.Request.URL.String()
	canonicalURL, err := normalizeURL(finalURL)
	if err != nil {
		return Result{}, err
	}

	return Result{
		FinalURL:     finalURL,
		CanonicalURL: canonicalURL,
	}, nil
}

func (r *GoogleNewsResolver) resolveGoogleNewsArticle(ctx context.Context, rawURL string) (Result, error) {
	token, err := extractGoogleNewsToken(rawURL)
	if err != nil {
		return Result{}, err
	}

	signature, timestamp, err := r.fetchDecodingParams(ctx, token)
	if err != nil {
		return Result{}, err
	}

	decodedURL, err := r.decodeGoogleNewsURL(ctx, token, timestamp, signature)
	if err != nil {
		return Result{}, err
	}

	canonicalURL, err := normalizeURL(decodedURL)
	if err != nil {
		return Result{}, err
	}

	return Result{
		FinalURL:     decodedURL,
		CanonicalURL: canonicalURL,
	}, nil
}

func (r *GoogleNewsResolver) fetchDecodingParams(ctx context.Context, token string) (string, string, error) {
	candidates := []string{
		"https://news.google.com/articles/" + token,
		"https://news.google.com/rss/articles/" + token,
	}

	for _, candidate := range candidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
		if err != nil {
			return "", "", fmt.Errorf("创建 Google News 参数请求失败: %w", err)
		}
		req.Header.Set("User-Agent", r.userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml")

		resp, err := r.httpClient.Do(req)
		if err != nil {
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		signature := findMatch(signaturePattern, string(body))
		timestamp := findMatch(timestampPattern, string(body))
		if signature != "" && timestamp != "" {
			return signature, timestamp, nil
		}
	}

	return "", "", fmt.Errorf("未找到 Google News 解码参数")
}

func (r *GoogleNewsResolver) decodeGoogleNewsURL(ctx context.Context, token, timestamp, signature string) (string, error) {
	fReq := fmt.Sprintf(
		`[[["Fbv4je","[\"garturlreq\",[[\"X\",\"X\",[\"X\",\"X\"],null,null,1,1,\"US:en\",null,1,null,null,null,null,null,0,1],\"X\",\"X\",1,[1,1,1],1,1,null,0,0,null,0],\"%s\",%s,\"%s\"]",null,"generic"]]]`,
		token,
		timestamp,
		signature,
	)

	form := url.Values{}
	form.Set("f.req", fReq)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://news.google.com/_/DotsSplashUi/data/batchexecute",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("创建 Google News 解码请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 Google News 解码接口失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Google News 解码响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google News 解码接口返回异常: %d", resp.StatusCode)
	}

	parts := strings.SplitN(string(body), "\n\n", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("Google News 解码响应格式异常")
	}

	var outer []any
	if err := json.Unmarshal([]byte(parts[1]), &outer); err != nil {
		return "", fmt.Errorf("解析 Google News 外层响应失败: %w", err)
	}
	if len(outer) == 0 {
		return "", fmt.Errorf("Google News 外层响应为空")
	}

	row, ok := outer[0].([]any)
	if !ok || len(row) < 3 {
		return "", fmt.Errorf("Google News 外层响应结构异常")
	}

	blob, ok := row[2].(string)
	if !ok || strings.TrimSpace(blob) == "" {
		return "", fmt.Errorf("Google News 解码结果缺少数据")
	}

	var inner []any
	if err := json.Unmarshal([]byte(blob), &inner); err != nil {
		return "", fmt.Errorf("解析 Google News 内层响应失败: %w", err)
	}
	if len(inner) < 2 {
		return "", fmt.Errorf("Google News 内层响应结构异常")
	}

	decodedURL, ok := inner[1].(string)
	if !ok || strings.TrimSpace(decodedURL) == "" {
		return "", fmt.Errorf("Google News 原站 URL 为空")
	}

	return html.UnescapeString(decodedURL), nil
}

func isGoogleNewsArticleURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsed.Hostname(), "news.google.com") {
		return false
	}

	path := strings.Trim(parsed.Path, "/")
	return strings.HasPrefix(path, "rss/articles/") || strings.HasPrefix(path, "articles/") || strings.HasPrefix(path, "read/")
}

func extractGoogleNewsToken(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("解析 Google News URL 失败: %w", err)
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("Google News URL 格式异常: %s", rawURL)
	}

	token := strings.TrimSpace(parts[len(parts)-1])
	if token == "" {
		return "", fmt.Errorf("Google News token 为空")
	}
	return token, nil
}

func findMatch(pattern *regexp.Regexp, text string) string {
	matches := pattern.FindStringSubmatch(text)
	if len(matches) != 2 {
		return ""
	}
	return html.UnescapeString(matches[1])
}

func normalizeURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("解析最终链接失败: %w", err)
	}

	parsed.Fragment = ""
	parsed.Host = strings.ToLower(parsed.Host)
	if (parsed.Scheme == "https" && parsed.Port() == "443") || (parsed.Scheme == "http" && parsed.Port() == "80") {
		parsed.Host = parsed.Hostname()
	}

	queryValues := parsed.Query()
	trackingKeys := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"gclid", "fbclid", "ocid", "ref", "ref_src",
	}
	for _, key := range trackingKeys {
		queryValues.Del(key)
	}
	parsed.RawQuery = queryValues.Encode()

	return parsed.String(), nil
}
