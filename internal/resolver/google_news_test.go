package resolver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"auto-search/internal/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestGoogleNewsResolverResolve(t *testing.T) {
	t.Parallel()

	resolver := NewGoogleNewsResolver(config.HTTPConfig{
		TimeoutSeconds: 10,
		UserAgent:      "auto-search-test",
		MaxRedirects:   5,
	})
	resolver.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			finalURL, _ := url.Parse("https://example.com/article?id=1&utm_source=test#part")
			finalReq := req.Clone(req.Context())
			finalReq.URL = finalURL

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("ok")),
				Header:     make(http.Header),
				Request:    finalReq,
			}, nil
		}),
	}

	result, err := resolver.Resolve(context.Background(), "https://example.test/redirect")
	if err != nil {
		t.Fatalf("解析跳转链接失败: %v", err)
	}

	if result.FinalURL != "https://example.com/article?id=1&utm_source=test#part" {
		t.Fatalf("期望 final_url 正确，实际为 %s", result.FinalURL)
	}
	if result.CanonicalURL != "https://example.com/article?id=1" {
		t.Fatalf("期望 canonical_url 去掉跟踪参数，实际为 %s", result.CanonicalURL)
	}
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	canonicalURL, err := normalizeURL("HTTPS://Example.com:443/article?id=1&utm_source=test#part")
	if err != nil {
		t.Fatalf("规范化 URL 失败: %v", err)
	}

	if canonicalURL != "https://example.com/article?id=1" {
		t.Fatalf("规范化结果不正确: %s", canonicalURL)
	}
}
