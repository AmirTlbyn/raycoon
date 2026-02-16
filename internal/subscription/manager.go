package subscription

import (
	"context"
	"fmt"
	"time"

	"raycoon/internal/config/parser"
	"raycoon/internal/storage"
	pkgerrors "raycoon/pkg/errors"
)

// Manager manages subscriptions and their updates
type Manager struct {
	storage storage.Storage
	fetcher *Fetcher
	decoder *Decoder
	parser  *parser.Registry
}

// NewManager creates a new subscription manager
func NewManager(store storage.Storage, parserRegistry *parser.Registry) *Manager {
	return &Manager{
		storage: store,
		fetcher: NewFetcher(DefaultFetcherConfig()),
		decoder: NewDecoder(),
		parser:  parserRegistry,
	}
}

// UpdateResult represents the result of a subscription update
type UpdateResult struct {
	GroupID       int64
	GroupName     string
	Added         int
	Removed       int
	Failed        int
	TotalURIs     int
	Errors        []error
	UpdatedAt     time.Time
}

// UpdateGroup updates a group's subscription
func (m *Manager) UpdateGroup(ctx context.Context, groupID int64) (*UpdateResult, error) {
	// Get group
	group, err := m.storage.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Check if group has subscription
	if group.SubscriptionURL == nil || *group.SubscriptionURL == "" {
		return nil, fmt.Errorf("group has no subscription URL")
	}

	result := &UpdateResult{
		GroupID:   groupID,
		GroupName: group.Name,
		UpdatedAt: time.Now(),
	}

	// Fetch subscription
	content, err := m.fetcher.Fetch(ctx, *group.SubscriptionURL)
	if err != nil {
		return nil, &pkgerrors.SubscriptionError{
			Name: group.Name,
			URL:  *group.SubscriptionURL,
			Err:  err,
		}
	}

	// Decode subscription
	uris, err := m.decoder.Decode(content)
	if err != nil {
		return nil, &pkgerrors.SubscriptionError{
			Name: group.Name,
			URL:  *group.SubscriptionURL,
			Err:  err,
		}
	}

	result.TotalURIs = len(uris)

	// Start transaction
	tx, err := m.storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Delete old subscription configs
	if err := tx.DeleteConfigsByGroup(ctx, groupID, true); err != nil {
		return nil, fmt.Errorf("failed to delete old configs: %w", err)
	}

	// Parse and add new configs
	for _, uri := range uris {
		config, err := m.parser.Parse(uri)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("failed to parse URI: %w", err))
			continue
		}

		// Set group and source
		config.GroupID = groupID
		config.FromSubscription = true
		config.Enabled = true

		// Create config
		if err := tx.CreateConfig(ctx, config); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("failed to save config '%s': %w", config.Name, err))
			continue
		}

		result.Added++
	}

	// Update group's last updated timestamp
	now := time.Now()
	group.LastUpdated = &now

	// Calculate next update time
	if group.AutoUpdate {
		nextUpdate := now.Add(time.Duration(group.UpdateInterval) * time.Second)
		group.NextUpdate = &nextUpdate
	}

	if err := tx.UpdateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// UpdateGroupByName updates a group's subscription by name
func (m *Manager) UpdateGroupByName(ctx context.Context, name string) (*UpdateResult, error) {
	group, err := m.storage.GetGroupByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return m.UpdateGroup(ctx, group.ID)
}

// UpdateAllDue updates all groups with subscriptions that are due for update
func (m *Manager) UpdateAllDue(ctx context.Context) ([]*UpdateResult, error) {
	// Get groups due for update
	groups, err := m.storage.GetDueGroups(ctx)
	if err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		return []*UpdateResult{}, nil
	}

	results := make([]*UpdateResult, 0, len(groups))

	for _, group := range groups {
		result, err := m.UpdateGroup(ctx, group.ID)
		if err != nil {
			// Continue with other groups even if one fails
			results = append(results, &UpdateResult{
				GroupID:   group.ID,
				GroupName: group.Name,
				Failed:    1,
				Errors:    []error{err},
				UpdatedAt: time.Now(),
			})
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// GetUpdateStatus returns the update status for all groups with subscriptions
func (m *Manager) GetUpdateStatus(ctx context.Context) ([]*GroupUpdateStatus, error) {
	groups, err := m.storage.GetAllGroups(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]*GroupUpdateStatus, 0)

	for _, group := range groups {
		// Skip groups without subscriptions
		if group.SubscriptionURL == nil || *group.SubscriptionURL == "" {
			continue
		}

		status := &GroupUpdateStatus{
			GroupID:     group.ID,
			GroupName:   group.Name,
			URL:         *group.SubscriptionURL,
			AutoUpdate:  group.AutoUpdate,
			Interval:    time.Duration(group.UpdateInterval) * time.Second,
			LastUpdated: group.LastUpdated,
			NextUpdate:  group.NextUpdate,
		}

		// Check if update is due
		if group.NextUpdate != nil {
			status.IsDue = time.Now().After(*group.NextUpdate)
		} else if group.LastUpdated == nil {
			// Never updated
			status.IsDue = true
		}

		// Count configs from this subscription
		filter := storage.ConfigFilter{
			GroupID:          &group.ID,
			FromSubscription: func() *bool { b := true; return &b }(),
		}
		configs, _ := m.storage.GetAllConfigs(ctx, filter)
		status.ConfigCount = len(configs)

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GroupUpdateStatus represents the update status of a subscription group
type GroupUpdateStatus struct {
	GroupID     int64
	GroupName   string
	URL         string
	AutoUpdate  bool
	Interval    time.Duration
	LastUpdated *time.Time
	NextUpdate  *time.Time
	IsDue       bool
	ConfigCount int
}
