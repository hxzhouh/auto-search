package scheduler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"auto-search/internal/cleaning"
	"auto-search/internal/config"
	"auto-search/internal/discovery"
	"auto-search/internal/extraction"
)

func TestRunCycleSequence(t *testing.T) {
	t.Parallel()

	calls := make([]string, 0, 3)
	loop := NewLoop(
		config.SchedulerConfig{
			DiscoverIntervalMinutes: 60,
			ExtractBatchSize:        1,
			CleanBatchSize:          1,
			IdleWaitSeconds:         1,
		},
		discoverFunc(func(ctx context.Context) (discovery.Stats, error) {
			calls = append(calls, "discover")
			return discovery.Stats{FeedItems: 1, Inserted: 1}, nil
		}),
		extractFunc(func(ctx context.Context, limit int) (extraction.Stats, error) {
			calls = append(calls, "extract")
			if limit != 1 {
				t.Fatalf("extract limit 错误: %d", limit)
			}
			return extraction.Stats{Selected: 1, Extracted: 1}, nil
		}),
		cleanFunc(func(ctx context.Context, limit int) (cleaning.Stats, error) {
			calls = append(calls, "clean")
			if limit != 1 {
				t.Fatalf("clean limit 错误: %d", limit)
			}
			return cleaning.Stats{Selected: 1, Cleaned: 1}, nil
		}),
	)
	loop.logf = func(string, ...any) {}
	loop.now = func() time.Time {
		return time.Unix(3600, 0)
	}

	nextDiscoverAt := time.Unix(0, 0)
	didWork := loop.runCycle(context.Background(), &nextDiscoverAt)
	if !didWork {
		t.Fatalf("期望本轮有工作")
	}

	expected := []string{"discover", "extract", "clean"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("执行顺序错误: got=%v want=%v", calls, expected)
	}
	if !nextDiscoverAt.Equal(time.Unix(3600, 0).Add(time.Hour)) {
		t.Fatalf("下次 discover 时间错误: %v", nextDiscoverAt)
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	loop := NewLoop(
		config.SchedulerConfig{
			DiscoverIntervalMinutes: 60,
			ExtractBatchSize:        1,
			CleanBatchSize:          1,
			IdleWaitSeconds:         1,
		},
		discoverFunc(func(ctx context.Context) (discovery.Stats, error) {
			t.Fatal("不应执行 discover")
			return discovery.Stats{}, nil
		}),
		extractFunc(func(ctx context.Context, limit int) (extraction.Stats, error) {
			t.Fatal("不应执行 extract")
			return extraction.Stats{}, nil
		}),
		cleanFunc(func(ctx context.Context, limit int) (cleaning.Stats, error) {
			t.Fatal("不应执行 clean")
			return cleaning.Stats{}, nil
		}),
	)
	loop.logf = func(string, ...any) {}

	err := loop.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("期望 context.Canceled，实际为 %v", err)
	}
}

type discoverFunc func(ctx context.Context) (discovery.Stats, error)

func (f discoverFunc) Run(ctx context.Context) (discovery.Stats, error) {
	return f(ctx)
}

type extractFunc func(ctx context.Context, limit int) (extraction.Stats, error)

func (f extractFunc) Run(ctx context.Context, limit int) (extraction.Stats, error) {
	return f(ctx, limit)
}

type cleanFunc func(ctx context.Context, limit int) (cleaning.Stats, error)

func (f cleanFunc) Run(ctx context.Context, limit int) (cleaning.Stats, error) {
	return f(ctx, limit)
}
