package tui

import (
	"raycoon/internal/core/types"
	"raycoon/internal/latency"
	"raycoon/internal/storage/models"
	"raycoon/internal/subscription"
)

// Data loading messages.

type groupsLoadedMsg struct {
	groups []*models.Group
	err    error
}

type configsLoadedMsg struct {
	configs []*models.Config
	err     error
}

type settingsLoadedMsg struct {
	settings map[string]string
	err      error
}

type activeConnLoadedMsg struct {
	conn   *models.ActiveConnection
	config *models.Config
	group  *models.Group
	err    error
}

// Connection lifecycle messages.

type connectStartedMsg struct{}

type connectResultMsg struct {
	config *models.Config
	err    error
}

type disconnectResultMsg struct {
	err error
}

// Status polling messages.

type statusTickMsg struct{}

type statusResultMsg struct {
	status  *types.Status
	stats   *types.Stats
	running bool
	err     error
}

// Latency testing messages.

type latencyTestStartMsg struct{}

type latencyTestProgressMsg struct {
	result  *latency.TestResult
	current int
	total   int
}

type latencyTestDoneMsg struct {
	batch *latency.BatchResult
	err   error
}

type singleLatencyDoneMsg struct {
	result *latency.TestResult
}

// Subscription update messages.

type subUpdateStartMsg struct{}

type subUpdateResultMsg struct {
	result *subscription.UpdateResult
	err    error
}

// Settings update messages.

type settingSavedMsg struct {
	key string
	err error
}

// Notification message.

type notificationMsg struct {
	text    string
	isError bool
}

type clearNotificationMsg struct {
	version int
}
