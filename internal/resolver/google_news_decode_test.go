package resolver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"auto-search/internal/config"
)

type decodeRoundTripFunc func(req *http.Request) (*http.Response, error)

func (fn decodeRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type roundTripSequence struct {
	handlers []func(req *http.Request) (*http.Response, error)
	index    int
}

func (r *roundTripSequence) RoundTrip(req *http.Request) (*http.Response, error) {
	handler := r.handlers[r.index]
	r.index++
	return handler(req)
}

func TestResolveGoogleNewsArticle(t *testing.T) {
	t.Parallel()

	transport := &roundTripSequence{
		handlers: []func(req *http.Request) (*http.Response, error){
			func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://news.google.com/articles/CBMiTEST" {
					t.Fatalf("首次请求 URL 错误: %s", req.URL.String())
				}
				body := `<c-wiz><div jscontroller="x" data-n-a-sg="SIG123" data-n-a-ts="1741600000"></div></c-wiz>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			},
			func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != "https://news.google.com/_/DotsSplashUi/data/batchexecute" {
					t.Fatalf("解码请求 URL 错误: %s", req.URL.String())
				}
				body := `)]}'` + "\n\n" + `[["wrb.fr","Fbv4je","[\"garturlres\",\"https://example.com/article?id=1&utm_source=test\"]",null,null,null,"generic"]]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			},
		},
	}

	resolver := NewGoogleNewsResolver(config.HTTPConfig{
		TimeoutSeconds: 10,
		UserAgent:      "auto-search-test",
		MaxRedirects:   5,
	})
	resolver.httpClient = &http.Client{Transport: transport}

	result, err := resolver.Resolve(context.Background(), "https://news.google.com/rss/articles/CBMiTEST?oc=5")
	if err != nil {
		t.Fatalf("解析 Google News 原站 URL 失败: %v", err)
	}

	if result.FinalURL != "https://example.com/article?id=1&utm_source=test" {
		t.Fatalf("final_url 错误: %s", result.FinalURL)
	}
	if result.CanonicalURL != "https://example.com/article?id=1" {
		t.Fatalf("canonical_url 错误: %s", result.CanonicalURL)
	}
}

func TestDecodeGoogleNewsURLParsesResponse(t *testing.T) {
	t.Parallel()

	resolver := NewGoogleNewsResolver(config.HTTPConfig{
		TimeoutSeconds: 10,
		UserAgent:      "auto-search-test",
		MaxRedirects:   5,
	})
	resolver.httpClient = &http.Client{
		Transport: decodeRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			body := `)]}'` + "\n\n" + `[["wrb.fr","Fbv4je","[\"garturlres\",\"https://example.com/path?a=1&amp;b=2\"]",null,null,null,"generic"]]`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	decodedURL, err := resolver.decodeGoogleNewsURL(context.Background(), "TOKEN", "123", "SIG")
	if err != nil {
		t.Fatalf("解码 Google News URL 失败: %v", err)
	}
	if decodedURL != "https://example.com/path?a=1&b=2" {
		t.Fatalf("解码结果错误: %s", decodedURL)
	}
}

func TestIsGoogleNewsArticleURL(t *testing.T) {
	t.Parallel()

	if !isGoogleNewsArticleURL("https://news.google.com/rss/articles/abc?oc=5") {
		t.Fatal("应识别为 Google News 文章链接")
	}
	if isGoogleNewsArticleURL("https://example.com/article") {
		t.Fatal("不应识别普通链接")
	}
}
