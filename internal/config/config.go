package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Config struct {
	App       AppConfig       `json:"app"`
	Web       WebConfig       `json:"web"`
	Scheduler SchedulerConfig `json:"scheduler"`
	Database  DatabaseConfig  `json:"database"`
	HTTP      HTTPConfig      `json:"http"`
	Defuddle  DefuddleConfig  `json:"defuddle"`
	AI        AIConfig        `json:"ai"`
}

type AppConfig struct {
	Name string `json:"name"`
	Env  string `json:"env"`
}

type WebConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type SchedulerConfig struct {
	DiscoverIntervalMinutes int `json:"discover_interval_minutes"`
	ExtractBatchSize        int `json:"extract_batch_size"`
	CleanBatchSize          int `json:"clean_batch_size"`
	IdleWaitSeconds         int `json:"idle_wait_seconds"`
}

type DatabaseConfig struct {
	Driver string       `json:"driver"`
	MySQL  MySQLConfig  `json:"mysql"`
	SQLite SQLiteConfig `json:"sqlite"`
}

type MySQLConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	User            string `json:"user"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	Params          string `json:"params"`
	MaxOpenConns    int    `json:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	ConnMaxLifetime int    `json:"conn_max_lifetime_seconds"`
}

type SQLiteConfig struct {
	Path string `json:"path"`
}

type HTTPConfig struct {
	TimeoutSeconds    int    `json:"timeout_seconds"`
	UserAgent         string `json:"user_agent"`
	MaxRedirects      int    `json:"max_redirects"`
	RequestIntervalMS int    `json:"request_interval_ms"`
}

type DefuddleConfig struct {
	BaseURL string `json:"base_url"`
}

type AIConfig struct {
	Provider       string  `json:"provider"`
	BaseURL        string  `json:"base_url"`
	APIKey         string  `json:"api_key"`
	Model          string  `json:"model"`
	MaxTokens      int     `json:"max_tokens"`
	Temperature    float64 `json:"temperature"`
	RPMLimit       int     `json:"rpm_limit"`
	TimeoutSeconds int     `json:"timeout_seconds"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.App.Name == "" {
		c.App.Name = "auto-search"
	}
	if c.App.Env == "" {
		c.App.Env = "local"
	}
	if c.Web.Host == "" {
		c.Web.Host = "0.0.0.0"
	}
	if c.Web.Port <= 0 {
		c.Web.Port = 8080
	}
	if c.Scheduler.DiscoverIntervalMinutes <= 0 {
		c.Scheduler.DiscoverIntervalMinutes = 60
	}
	if c.Scheduler.ExtractBatchSize <= 0 {
		c.Scheduler.ExtractBatchSize = 1
	}
	if c.Scheduler.CleanBatchSize <= 0 {
		c.Scheduler.CleanBatchSize = 1
	}
	if c.Scheduler.IdleWaitSeconds <= 0 {
		c.Scheduler.IdleWaitSeconds = 10
	}
	if c.HTTP.TimeoutSeconds <= 0 {
		c.HTTP.TimeoutSeconds = 20
	}
	if c.HTTP.UserAgent == "" {
		c.HTTP.UserAgent = "auto-search/0.1"
	}
	if c.HTTP.MaxRedirects <= 0 {
		c.HTTP.MaxRedirects = 10
	}
	if c.HTTP.RequestIntervalMS < 0 {
		c.HTTP.RequestIntervalMS = 0
	}
	if c.Defuddle.BaseURL == "" {
		c.Defuddle.BaseURL = "http://defuddle.md"
	}
	if c.AI.MaxTokens <= 0 {
		c.AI.MaxTokens = 1200
	}
	if c.AI.Temperature == 0 {
		c.AI.Temperature = 0.2
	}
	if c.AI.RPMLimit < 0 {
		c.AI.RPMLimit = 0
	}
	if c.AI.TimeoutSeconds <= 0 {
		c.AI.TimeoutSeconds = 120
	}

	switch c.Database.Driver {
	case "mysql":
		return validateMySQL(c.Database.MySQL)
	case "sqlite":
		if c.Database.SQLite.Path == "" {
			return errors.New("sqlite.path 不能为空")
		}
		return nil
	default:
		return fmt.Errorf("不支持的数据库驱动: %s", c.Database.Driver)
	}
}

func (c Config) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Web.Host, c.Web.Port)
}

func validateMySQL(cfg MySQLConfig) error {
	switch {
	case cfg.Host == "":
		return errors.New("mysql.host 不能为空")
	case cfg.Port <= 0:
		return errors.New("mysql.port 必须大于 0")
	case cfg.User == "":
		return errors.New("mysql.user 不能为空")
	case cfg.Database == "":
		return errors.New("mysql.database 不能为空")
	default:
		return nil
	}
}
