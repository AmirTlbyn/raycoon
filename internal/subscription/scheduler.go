package subscription

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// Scheduler handles automatic subscription updates
type Scheduler struct {
	scheduler gocron.Scheduler
	manager   *Manager
	running   bool
}

// NewScheduler creates a new subscription scheduler
func NewScheduler(manager *Manager) (*Scheduler, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %w", err)
	}

	return &Scheduler{
		scheduler: scheduler,
		manager:   manager,
	}, nil
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	// Add periodic job to check for due updates
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(5*time.Minute), // Check every 5 minutes
		gocron.NewTask(func() {
			s.checkAndUpdateDue(ctx)
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create update job: %w", err)
	}

	// Start the scheduler
	s.scheduler.Start()
	s.running = true

	// Run initial check
	go s.checkAndUpdateDue(ctx)

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	if err := s.scheduler.Shutdown(); err != nil {
		return fmt.Errorf("failed to stop scheduler: %w", err)
	}

	s.running = false
	return nil
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	return s.running
}

// checkAndUpdateDue checks for due subscriptions and updates them
func (s *Scheduler) checkAndUpdateDue(ctx context.Context) {
	results, err := s.manager.UpdateAllDue(ctx)
	if err != nil {
		log.Printf("Error checking due subscriptions: %v", err)
		return
	}

	// Log results
	for _, result := range results {
		if len(result.Errors) > 0 {
			log.Printf("Subscription update failed for group '%s': added=%d, failed=%d, errors=%v",
				result.GroupName, result.Added, result.Failed, result.Errors)
		} else {
			log.Printf("Subscription updated for group '%s': added=%d configs",
				result.GroupName, result.Added)
		}
	}
}

// ForceUpdateAll forces an update of all subscription groups
func (s *Scheduler) ForceUpdateAll(ctx context.Context) ([]*UpdateResult, error) {
	return s.manager.UpdateAllDue(ctx)
}

// ScheduleCustomJob schedules a custom job
func (s *Scheduler) ScheduleCustomJob(interval time.Duration, task func()) error {
	_, err := s.scheduler.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(task),
	)
	return err
}
