package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"auto-search/internal/config"
)

type DefuddleClient struct {
	bin string
}

type Article struct {
	Title       string
	Author      string
	Description string
	PublishedAt *time.Time
	Markdown    string
	RawMarkdown string
}

type defuddleOutput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Published   string `json:"published"`
	Content     string `json:"content"`
}

func NewDefuddleClient(cfg *config.Config) *DefuddleClient {
	return &DefuddleClient{bin: cfg.Defuddle.Bin}
}

func (c *DefuddleClient) Extract(ctx context.Context, targetURL string) (Article, error) {
	cmd := exec.CommandContext(ctx, c.bin, "parse", "--json", targetURL)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if ok := isExitError(err, &exitErr); ok {
			return Article{}, fmt.Errorf("defuddle 执行失败 (exit %d): %s", exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return Article{}, fmt.Errorf("defuddle 执行失败: %w", err)
	}

	var result defuddleOutput
	if err := json.Unmarshal(out, &result); err != nil {
		return Article{}, fmt.Errorf("解析 defuddle 输出失败: %w", err)
	}

	if strings.TrimSpace(result.Content) == "" {
		return Article{}, fmt.Errorf("defuddle 返回内容为空")
	}

	article := Article{
		Title:       result.Title,
		Author:      result.Author,
		Description: result.Description,
		Markdown:    result.Content,
		RawMarkdown: string(out),
	}

	if result.Published != "" {
		article.PublishedAt = parsePublishedAt(result.Published)
	}

	return article, nil
}

func isExitError(err error, target **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if ok {
		*target = e
	}
	return ok
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
