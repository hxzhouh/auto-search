package scheduler

import (
	"context"
	"fmt"
	"time"

	"auto-search/internal/cleaning"
	"auto-search/internal/config"
	"auto-search/internal/discovery"
	"auto-search/internal/extraction"
)

type discoverer interface {
	Run(ctx context.Context) (discovery.Stats, error)
}

type extractor interface {
	Run(ctx context.Context, limit int) (extraction.Stats, error)
}

type cleaner interface {
	Run(ctx context.Context, limit int) (cleaning.Stats, error)
}

type Loop struct {
	cfg      config.SchedulerConfig
	discover discoverer
	extract  extractor
	clean    cleaner
	now      func() time.Time
	sleep    func(context.Context, time.Duration) bool
	logf     func(format string, args ...any)
}

func NewLoop(cfg config.SchedulerConfig, discover discoverer, extract extractor, clean cleaner) *Loop {
	return &Loop{
		cfg:      cfg,
		discover: discover,
		extract:  extract,
		clean:    clean,
		now:      time.Now,
		sleep:    sleepContext,
		logf: func(format string, args ...any) {
			fmt.Printf(format+"\n", args...)
		},
	}
}

func (l *Loop) Run(ctx context.Context) error {
	nextDiscoverAt := l.now()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		didWork := l.runCycle(ctx, &nextDiscoverAt)
		if didWork {
			continue
		}

		waitFor := time.Until(nextDiscoverAt)
		idleWait := time.Duration(l.cfg.IdleWaitSeconds) * time.Second
		if waitFor <= 0 || waitFor > idleWait {
			waitFor = idleWait
		}
		if !l.sleep(ctx, waitFor) {
			return ctx.Err()
		}
	}
}

func (l *Loop) runCycle(ctx context.Context, nextDiscoverAt *time.Time) bool {
	didWork := false

	if !l.now().Before(*nextDiscoverAt) {
		stats, err := l.discover.Run(ctx)
		if err != nil {
			l.logf("scheduler discover 失败: %v", err)
		} else {
			l.logf(
				"scheduler discover 完成: queries=%d feed_items=%d inserted=%d url_duplicates=%d title_duplicates=%d resolve_failures=%d fetch_failures=%d",
				stats.Queries,
				stats.FeedItems,
				stats.Inserted,
				stats.URLDuplicates,
				stats.TitleDuplicates,
				stats.ResolveFailures,
				stats.FetchFailures,
			)
			didWork = didWork || stats.FeedItems > 0 || stats.Inserted > 0
		}

		*nextDiscoverAt = l.now().Add(time.Duration(l.cfg.DiscoverIntervalMinutes) * time.Minute)
	}

	extractStats, err := l.extract.Run(ctx, l.cfg.ExtractBatchSize)
	if err != nil {
		l.logf("scheduler extract 失败: %v", err)
	} else {
		if extractStats.Selected > 0 {
			l.logf(
				"scheduler extract 完成: selected=%d extracted=%d failed=%d",
				extractStats.Selected,
				extractStats.Extracted,
				extractStats.Failed,
			)
		}
		didWork = didWork || extractStats.Selected > 0 || extractStats.Extracted > 0
	}

	cleanStats, err := l.clean.Run(ctx, l.cfg.CleanBatchSize)
	if err != nil {
		l.logf("scheduler clean 失败: %v", err)
	} else {
		if cleanStats.Selected > 0 {
			l.logf(
				"scheduler clean 完成: selected=%d cleaned=%d failed=%d",
				cleanStats.Selected,
				cleanStats.Cleaned,
				cleanStats.Failed,
			)
		}
		didWork = didWork || cleanStats.Selected > 0 || cleanStats.Cleaned > 0
	}

	return didWork
}

func sleepContext(ctx context.Context, waitFor time.Duration) bool {
	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
