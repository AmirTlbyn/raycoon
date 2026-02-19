package tui

import (
	"context"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"raycoon/internal/core"
	"raycoon/internal/core/tun"
	"raycoon/internal/core/types"
	"raycoon/internal/latency"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
	"raycoon/internal/subscription"
)

// loadGroups fetches all groups with their config counts.
func loadGroups(store storage.Storage) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		groups, err := store.GetAllGroups(ctx)
		return groupsLoadedMsg{groups: groups, err: err}
	}
}

// loadConfigs fetches configs, optionally filtered by group.
func loadConfigs(store storage.Storage, groupID *int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		filter := storage.ConfigFilter{GroupID: groupID}
		configs, err := store.GetAllConfigs(ctx, filter)
		return configsLoadedMsg{configs: configs, err: err}
	}
}

// loadSettings fetches all application settings.
func loadSettings(store storage.Storage) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		settings, err := store.GetAllSettings(ctx)
		return settingsLoadedMsg{settings: settings, err: err}
	}
}

// loadActiveConnection loads the current active connection with its config and group.
func loadActiveConnection(store storage.Storage) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		conn, err := store.GetActiveConnection(ctx)
		if err != nil || conn == nil {
			return activeConnLoadedMsg{}
		}
		config, err := store.GetConfig(ctx, conn.ConfigID)
		if err != nil {
			return activeConnLoadedMsg{conn: conn, err: err}
		}
		group, _ := store.GetGroup(ctx, config.GroupID)
		return activeConnLoadedMsg{conn: conn, config: config, group: group}
	}
}

// connectToConfig starts the proxy core with the given config.
func connectToConfig(store storage.Storage, mgr *core.Manager, config *models.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Read settings for ports/mode.
		socksPort := 1080
		httpPort := 1081
		vpnMode := types.VPNModeProxy

		if v, err := store.GetSetting(ctx, "proxy_port"); err == nil {
			if p, e := strconv.Atoi(v); e == nil {
				socksPort = p
			}
		}
		if v, err := store.GetSetting(ctx, "http_proxy_port"); err == nil {
			if p, e := strconv.Atoi(v); e == nil {
				httpPort = p
			}
		}
		if v, err := store.GetSetting(ctx, "vpn_mode"); err == nil {
			vpnMode = types.VPNMode(v)
		}

		coreConfig := core.BuildCoreConfig(config, vpnMode, socksPort, httpPort)
		if err := mgr.Start(ctx, coreConfig); err != nil {
			return connectResultMsg{config: config, err: err}
		}

		// Enable TUN device for tunnel mode.
		if vpnMode == types.VPNModeTunnel {
			if err := tun.Enable(socksPort, []string{config.Address}); err != nil {
				mgr.Stop(ctx)
				return connectResultMsg{config: config, err: err}
			}
		}

		// Save active connection.
		activeConn := &models.ActiveConnection{
			ConfigID: config.ID,
			CoreType: string(mgr.GetCoreType()),
			VPNMode:  string(vpnMode),
		}
		store.SetActiveConnection(ctx, activeConn)

		// Update usage stats.
		now := time.Now()
		config.LastUsed = &now
		config.UseCount++
		store.UpdateConfig(ctx, config)

		return connectResultMsg{config: config}
	}
}

// disconnect stops the proxy core and removes system proxy if tunnel mode.
func disconnect(store storage.Storage, mgr *core.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Check if tunnel mode — need to disable TUN device.
		if conn, err := store.GetActiveConnection(ctx); err == nil && conn != nil {
			if conn.VPNMode == string(types.VPNModeTunnel) {
				tun.Disable()
			}
		}

		var err error
		if mgr.IsRunning() {
			err = mgr.Stop(ctx)
		}
		if err == nil {
			store.ClearActiveConnection(ctx)
		}
		return disconnectResultMsg{err: err}
	}
}

// pollStatus fetches core status and stats.
func pollStatus(mgr *core.Manager) tea.Cmd {
	return func() tea.Msg {
		running := mgr.IsRunning()
		if !running {
			return statusResultMsg{running: false}
		}
		status, _ := mgr.GetStatus()
		stats, _ := mgr.GetStats()
		return statusResultMsg{status: status, stats: stats, running: true}
	}
}

// statusTick returns a tea.Cmd that fires after 2 seconds.
func statusTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return statusTickMsg{}
	})
}

// testSingleLatency tests a single config.
func testSingleLatency(store storage.Storage, config *models.Config, strategyName string, timeoutMS int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		strategy, err := latency.NewStrategy(strategyName)
		if err != nil {
			strategy, _ = latency.NewStrategy("tcp")
		}
		tester := latency.NewTester(store, latency.TesterConfig{
			Workers:  1,
			Timeout:  time.Duration(timeoutMS) * time.Millisecond,
			Strategy: strategy,
		})
		result := tester.TestSingle(ctx, config)
		return singleLatencyDoneMsg{result: result}
	}
}

// testBatchLatency tests multiple configs with progress reporting via program.Send.
func testBatchLatency(store storage.Storage, configs []*models.Config, p *tea.Program, strategyName string, workers int64, timeoutMS int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		strategy, err := latency.NewStrategy(strategyName)
		if err != nil {
			strategy, _ = latency.NewStrategy("tcp")
		}
		tester := latency.NewTester(store, latency.TesterConfig{
			Workers:  workers,
			Timeout:  time.Duration(timeoutMS) * time.Millisecond,
			Strategy: strategy,
		})

		progress := func(result *latency.TestResult, current, total int) {
			p.Send(latencyTestProgressMsg{result: result, current: current, total: total})
		}

		batch := tester.TestBatch(ctx, configs, progress)
		return latencyTestDoneMsg{batch: batch}
	}
}

// updateSubscriptionWithManager triggers a subscription update using the manager directly.
func updateSubscriptionWithManager(subMgr *subscription.Manager, groupID int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result, err := subMgr.UpdateGroup(ctx, groupID)
		return subUpdateResultMsg{result: result, err: err}
	}
}

// saveSetting saves a single setting.
func saveSetting(store storage.Storage, key, value string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := store.SetSetting(ctx, key, value)
		return settingSavedMsg{key: key, err: err}
	}
}

// clearNotification returns a command that fires after a delay.
func clearNotification(d time.Duration, version int) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearNotificationMsg{version: version}
	})
}

// errNeedSubManager is a sentinel; we route subscription updates through the Manager directly.
var errNeedSubManager = errSentinel("subscription manager required")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
