package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"auto-search/internal/cleaning"
	"auto-search/internal/config"
	"auto-search/internal/database"
	"auto-search/internal/discovery"
	"auto-search/internal/extraction"
	"auto-search/internal/query"
	"auto-search/internal/webui"
)

const defaultConfigPath = "configs/config.local.json"

func Run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "migrate":
		return runMigrate(args[1:])
	case "discover":
		return runDiscover(args[1:])
	case "clean":
		return runClean(args[1:])
	case "extract":
		return runExtract(args[1:])
	case "serve":
		return runServe(args[1:])
	case "queries":
		return runQueries(args[1:])
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("不支持的命令: %s", args[0])
	}
}

func runMigrate(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", defaultConfigPath, "配置文件路径")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("解析 migrate 参数失败: %w", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.HTTP.TimeoutSeconds)*time.Second)
	defer cancel()

	if err := database.RunMigrations(ctx, db, cfg.Database.Driver); err != nil {
		return err
	}

	fmt.Printf("迁移完成，数据库驱动: %s\n", cfg.Database.Driver)
	return nil
}

func runQueries(args []string) error {
	if len(args) == 0 || args[0] != "list" {
		return fmt.Errorf("queries 目前只支持子命令: list")
	}

	fs := flag.NewFlagSet("queries list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", defaultConfigPath, "配置文件路径")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("解析 queries 参数失败: %w", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.HTTP.TimeoutSeconds)*time.Second)
	defer cancel()

	repo := query.NewRepository(db)
	items, err := repo.ListEnabled(ctx)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Println("当前没有启用的 query")
		return nil
	}

	for _, item := range items {
		fmt.Printf("[%d] %s | %s | %s/%s | priority=%d\n",
			item.ID,
			item.Name,
			item.QueryText,
			item.Lang,
			item.Region,
			item.Priority,
		)
	}

	return nil
}

func runDiscover(args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("discover 目前只支持子命令: run")
	}

	fs := flag.NewFlagSet("discover run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", defaultConfigPath, "配置文件路径")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("解析 discover 参数失败: %w", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	service := discovery.NewService(db, cfg)
	stats, err := service.Run(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("discover 完成: queries=%d feed_items=%d inserted=%d url_duplicates=%d title_duplicates=%d resolve_failures=%d fetch_failures=%d\n",
		stats.Queries,
		stats.FeedItems,
		stats.Inserted,
		stats.URLDuplicates,
		stats.TitleDuplicates,
		stats.ResolveFailures,
		stats.FetchFailures,
	)

	return nil
}

func runExtract(args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("extract 目前只支持子命令: run")
	}

	fs := flag.NewFlagSet("extract run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", defaultConfigPath, "配置文件路径")
	limit := fs.Int("limit", 20, "单次提取数量上限")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("解析 extract 参数失败: %w", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	service := extraction.NewService(db, cfg)
	stats, err := service.Run(context.Background(), *limit)
	if err != nil {
		return err
	}

	fmt.Printf("extract 完成: selected=%d extracted=%d failed=%d\n",
		stats.Selected,
		stats.Extracted,
		stats.Failed,
	)
	return nil
}

func runClean(args []string) error {
	if len(args) == 0 || args[0] != "run" {
		return fmt.Errorf("clean 目前只支持子命令: run")
	}

	fs := flag.NewFlagSet("clean run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", defaultConfigPath, "配置文件路径")
	limit := fs.Int("limit", 10, "单次清洗数量上限")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("解析 clean 参数失败: %w", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	service, err := cleaning.NewService(db, cfg)
	if err != nil {
		return err
	}

	stats, err := service.Run(context.Background(), *limit)
	if err != nil {
		return err
	}

	fmt.Printf("clean 完成: selected=%d cleaned=%d failed=%d\n",
		stats.Selected,
		stats.Cleaned,
		stats.Failed,
	)
	return nil
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	configPath := fs.String("config", defaultConfigPath, "配置文件路径")
	addr := fs.String("addr", ":8080", "监听地址")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("解析 serve 参数失败: %w", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Printf("前端页面已启动: %s\n", displayURL(*addr))
	return webui.NewServer(db).Serve(*addr)
}

func displayURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		return fmt.Sprintf("http://%s:%s", host, port)
	}
	if len(addr) > 0 && addr[0] == ':' {
		return "http://127.0.0.1" + addr
	}
	return "http://" + addr
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "用法:")
	fmt.Fprintln(w, "  auto-search migrate [-config configs/config.local.json]")
	fmt.Fprintln(w, "  auto-search discover run [-config configs/config.local.json]")
	fmt.Fprintln(w, "  auto-search extract run [-config configs/config.local.json] [-limit 20]")
	fmt.Fprintln(w, "  auto-search clean run [-config configs/config.local.json] [-limit 10]")
	fmt.Fprintln(w, "  auto-search serve [-config configs/config.local.json] [-addr :8080]")
	fmt.Fprintln(w, "  auto-search queries list [-config configs/config.local.json]")
}
