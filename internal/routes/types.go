package routes

// AlertRoute defines how alerts should be routed to notification channels
type AlertRoute struct {
	ID          string      `yaml:"id" json:"id"`                       // Unique identifier
	Name        string      `yaml:"name" json:"name"`                   // Display name
	Enabled     bool        `yaml:"enabled" json:"enabled"`             // Whether the route is active
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`

	// Matching conditions (all must match for AND logic)
	Match RouteMatch `yaml:"match" json:"match"`

	// Target channels (send to all if matched)
	ChannelIDs []string `yaml:"channel_ids" json:"channel_ids"` // e.g., ["wechat-ops", "dingtalk-oncall"]

	// Continue routing after match
	Continue bool `yaml:"continue,omitempty" json:"continue,omitempty"` // If true, continue matching next routes

	// Priority (higher evaluated first)
	Priority int `yaml:"priority,omitempty" json:"priority,omitempty"` // Default 0

	// Timestamps
	CreatedAt string `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt string `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// RouteMatch defines conditions for matching alerts
type RouteMatch struct {
	// Label matching (all labels must match)
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"` // Exact match: {"severity": "critical"}

	// Label regex matching
	LabelRegex map[string]string `yaml:"label_regex,omitempty" json:"label_regex,omitempty"` // Regex match

	// Severity matching
	Severities []string `yaml:"severities,omitempty" json:"severities,omitempty"` // ["critical", "warning"]

	// Alert name patterns
	AlertNames     string `yaml:"alert_names,omitempty" json:"alert_names,omitempty"`         // Exact match
	AlertNameRegex string `yaml:"alert_name_regex,omitempty" json:"alert_name_regex,omitempty"` // Regex match

	// Metric name patterns
	MetricNames     string `yaml:"metric_names,omitempty" json:"metric_names,omitempty"`         // Exact match
	MetricNameRegex string `yaml:"metric_name_regex,omitempty" json:"metric_name_regex,omitempty"` // Regex match
}

// NotificationChannel represents a single notification channel (e.g., a specific webhook)
type NotificationChannel struct {
	ID   string `yaml:"id" json:"id"` // Unique ID: "wechat-ops", "dingtalk-oncall"
	Name string `yaml:"name" json:"name"`
	Type string `yaml:"type" json:"type"` // "wechat", "dingtalk", "feishu", "sms", "call"

	// Type-specific configuration
	WeChat   interface{} `yaml:"wechat,omitempty" json:"wechat,omitempty"`
	DingTalk interface{} `yaml:"dingtalk,omitempty" json:"dingtalk,omitempty"`
	Feishu   interface{} `yaml:"feishu,omitempty" json:"feishu,omitempty"`

	// For future expansion (SMS, phone call)
	SMS       interface{} `yaml:"sms,omitempty" json:"sms,omitempty"`
	PhoneCall interface{} `yaml:"phone_call,omitempty" json:"phone_call,omitempty"`

	// Description and grouping
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"` // e.g., {"team": "ops"}

	// Status
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Timestamps
	CreatedAt string `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt string `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}
