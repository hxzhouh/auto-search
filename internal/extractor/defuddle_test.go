package extractor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"auto-search/internal/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestBuildDefuddleURL(t *testing.T) {
	t.Parallel()

	result, err := buildDefuddleURL("http://defuddle.md", "https://example.com/article/path?id=1")
	if err != nil {
		t.Fatalf("构造 defuddle URL 失败: %v", err)
	}

	if result != "http://defuddle.md/example.com/article/path?id=1" {
		t.Fatalf("defuddle URL 不正确: %s", result)
	}
}

func TestParseMarkdownDocument(t *testing.T) {
	t.Parallel()

	raw := `---
title: OpenAI 发布新模型
author: Alice
description: 一段摘要
published: 2026-03-10T12:00:00Z
---
# 正文

这里是正文。
`

	article, err := parseMarkdownDocument(raw)
	if err != nil {
		t.Fatalf("解析 markdown 文档失败: %v", err)
	}

	if article.Title != "OpenAI 发布新模型" {
		t.Fatalf("标题解析错误: %s", article.Title)
	}
	if article.Author != "Alice" {
		t.Fatalf("作者解析错误: %s", article.Author)
	}
	if article.Description != "一段摘要" {
		t.Fatalf("摘要解析错误: %s", article.Description)
	}
	if article.PublishedAt == nil || !article.PublishedAt.Equal(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("发布时间解析错误: %+v", article.PublishedAt)
	}
	if article.Markdown != "# 正文\n\n这里是正文。" {
		t.Fatalf("正文解析错误: %q", article.Markdown)
	}
}

func TestDefuddleClientExtract(t *testing.T) {
	t.Parallel()

	client := NewDefuddleClient(&config.Config{
		HTTP: config.HTTPConfig{
			TimeoutSeconds: 10,
			UserAgent:      "auto-search-test",
			MaxRedirects:   5,
		},
		Defuddle: config.DefuddleConfig{
			BaseURL: "http://defuddle.md",
		},
	})
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "http://defuddle.md/example.com/article" {
				t.Fatalf("请求 URL 不正确: %s", req.URL.String())
			}

			body := `---
title: Test
---
正文`

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	article, err := client.Extract(context.Background(), "https://example.com/article")
	if err != nil {
		t.Fatalf("调用 defuddle 失败: %v", err)
	}

	if article.Title != "Test" || article.Markdown != "正文" {
		t.Fatalf("提取结果不正确: %+v", article)
	}
}
