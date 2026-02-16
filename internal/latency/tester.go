package latency

import (
	"context"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
)

// TestResult holds the outcome for a single config test.
type TestResult struct {
	Config  *models.Config
	Latency *models.LatencyTest
}

// BatchResult holds the outcome of testing multiple configs.
type BatchResult struct {
	Results   []*TestResult
	Tested    int
	Succeeded int
	Failed    int
	Duration  time.Duration
}

// ProgressFunc is called each time a single test completes during batch testing.
type ProgressFunc func(result *TestResult, current, total int)

// TesterConfig holds configuration for the Tester.
type TesterConfig struct {
	Workers  int64
	Timeout  time.Duration
	Strategy Strategy
}

// Tester orchestrates latency testing.
type Tester struct {
	storage storage.Storage
	config  TesterConfig
}

// NewTester creates a new Tester.
func NewTester(store storage.Storage, cfg TesterConfig) *Tester {
	if cfg.Workers <= 0 {
		cfg.Workers = 10
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Tester{
		storage: store,
		config:  cfg,
	}
}

// TestSingle tests a single config and records the result.
func (t *Tester) TestSingle(ctx context.Context, config *models.Config) *TestResult {
	result := &TestResult{Config: config}

	testCtx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	latencyMS, err := t.config.Strategy.Test(testCtx, config)

	latencyTest := &models.LatencyTest{
		ConfigID:     config.ID,
		TestStrategy: t.config.Strategy.Name(),
		TestedAt:     time.Now(),
	}

	if err != nil {
		latencyTest.Success = false
		latencyTest.ErrorMessage = err.Error()
	} else {
		latencyTest.Success = true
		latencyTest.LatencyMS = &latencyMS
	}

	// Record to database (best-effort)
	t.storage.RecordLatency(ctx, latencyTest)

	result.Latency = latencyTest
	return result
}

// TestBatch tests multiple configs concurrently using a semaphore-based worker pool.
func (t *Tester) TestBatch(ctx context.Context, configs []*models.Config, progress ProgressFunc) *BatchResult {
	startTime := time.Now()

	batch := &BatchResult{}
	results := make([]*TestResult, len(configs))
	var mu sync.Mutex
	var completed int

	sem := semaphore.NewWeighted(t.config.Workers)
	var wg sync.WaitGroup

	for i, config := range configs {
		wg.Add(1)
		go func(idx int, cfg *models.Config) {
			defer wg.Done()

			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			defer sem.Release(1)

			result := t.TestSingle(ctx, cfg)
			results[idx] = result

			mu.Lock()
			completed++
			current := completed
			if result.Latency.Success {
				batch.Succeeded++
			} else {
				batch.Failed++
			}
			mu.Unlock()

			if progress != nil {
				progress(result, current, len(configs))
			}
		}(i, config)
	}

	wg.Wait()

	for _, r := range results {
		if r != nil {
			batch.Results = append(batch.Results, r)
			batch.Tested++
		}
	}

	// Sort: successful by latency ascending, failures at end
	sort.Slice(batch.Results, func(i, j int) bool {
		ri, rj := batch.Results[i].Latency, batch.Results[j].Latency
		if ri.Success && !rj.Success {
			return true
		}
		if !ri.Success && rj.Success {
			return false
		}
		if ri.Success && rj.Success {
			return *ri.LatencyMS < *rj.LatencyMS
		}
		return false
	})

	batch.Duration = time.Since(startTime)
	return batch
}
