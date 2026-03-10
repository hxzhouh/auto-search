package httpclient

import (
	"fmt"
	"net/http"
	"time"

	"auto-search/internal/config"
)

func New(cfg config.HTTPConfig) *http.Client {
	return &http.Client{
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("重定向次数超过限制: %d", cfg.MaxRedirects)
			}
			return nil
		},
	}
}
