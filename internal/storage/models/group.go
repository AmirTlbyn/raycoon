package models

import "time"

// Group represents a subscription group that can contain configs from a subscription link or manually added
type Group struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description,omitempty"`
	IsGlobal        bool       `json:"is_global"`
	SubscriptionURL *string    `json:"subscription_url,omitempty"` // Optional subscription link
	AutoUpdate      bool       `json:"auto_update"`
	UpdateInterval  int        `json:"update_interval"` // seconds
	LastUpdated     *time.Time `json:"last_updated,omitempty"`
	NextUpdate      *time.Time `json:"next_update,omitempty"`
	UserAgent       string     `json:"user_agent,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
