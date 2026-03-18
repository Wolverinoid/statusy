package models

import (
	"time"

	"gorm.io/gorm"
)

// ─── User ────────────────────────────────────────────────────────────────────

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type AuthSource string

const (
	AuthLocal AuthSource = "local"
	AuthLDAP  AuthSource = "ldap"
)

type User struct {
	gorm.Model
	Username   string     `gorm:"uniqueIndex;not null"`
	Email      string     `gorm:"uniqueIndex;not null"`
	Password   string     // bcrypt hash; empty for LDAP users
	DisplayName string
	Role       UserRole   `gorm:"default:user"`
	AuthSource AuthSource `gorm:"default:local"`
	Active     bool       `gorm:"default:true"`
}

// JWTBlacklist stores invalidated JWT IDs (jti) for session revocation.
type JWTBlacklist struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	JTI       string    `gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index"`
}

// ─── Monitor ─────────────────────────────────────────────────────────────────

type MonitorType string

const (
	MonitorHTTP         MonitorType = "http"
	MonitorPort         MonitorType = "port"
	MonitorPing         MonitorType = "ping"
	MonitorKeyword      MonitorType = "keyword"
	MonitorJSONAPI      MonitorType = "json_api"
	MonitorUDP          MonitorType = "udp"
	MonitorResponseTime MonitorType = "response_time"
	MonitorDNS          MonitorType = "dns"
	MonitorSSL          MonitorType = "ssl"
	MonitorDomainExpiry MonitorType = "domain_expiry"
)

type MonitorStatus string

const (
	StatusUp          MonitorStatus = "UP"
	StatusDown        MonitorStatus = "DOWN"
	StatusPending     MonitorStatus = "PENDING"
	StatusMaintenance MonitorStatus = "MAINTENANCE"
)

type Monitor struct {
	gorm.Model
	Name     string        `gorm:"not null"`
	Type     MonitorType   `gorm:"not null"`
	Status   MonitorStatus `gorm:"default:PENDING"`
	Active   bool          `gorm:"default:true"`

	// Scheduling
	IntervalSeconds int `gorm:"default:60"`
	TimeoutSeconds  int `gorm:"default:30"`
	Retries         int `gorm:"default:3"`

	// HTTP / Keyword / JSON / ResponseTime
	URL        string
	Method     string // GET | POST | HEAD
	Headers    string // JSON map[string]string
	Body       string
	ExpectedStatus int

	// Keyword
	Keyword   string
	KeywordMode string // contains | not_contains

	// JSON API
	JSONPath      string
	JSONExpected  string

	// Port / UDP
	Host string
	Port int

	// DNS
	DNSHost       string
	DNSRecordType string // A | AAAA | CNAME | MX | TXT | NS
	DNSExpected   string

	// SSL / Domain expiry
	Domain          string
	WarnDaysExpiry  int `gorm:"default:30"`

	// Response time threshold (ms)
	MaxResponseTimeMs int

	// Consecutive failures counter (for noise reduction)
	ConsecutiveFailures int `gorm:"default:0"`

	// Relations
	CheckResults  []CheckResult  `gorm:"foreignKey:MonitorID"`
	Notifications []Notification `gorm:"many2many:monitor_notifications"`
}

// ─── CheckResult ─────────────────────────────────────────────────────────────

type CheckResult struct {
	gorm.Model
	MonitorID      uint
	Status         MonitorStatus
	ResponseTimeMs int64
	Message        string // error message or details
	CheckedAt      time.Time
}

// ─── Notification Channel ────────────────────────────────────────────────────

type NotificationType string

const (
	NotifEmail    NotificationType = "email"
	NotifTelegram NotificationType = "telegram"
	NotifSlack    NotificationType = "slack"
)

type Notification struct {
	gorm.Model
	Name    string           `gorm:"not null"`
	Type    NotificationType `gorm:"not null"`
	Config  string           // JSON blob with channel-specific settings
	Active  bool             `gorm:"default:true"`

	// Alert rules
	NotifyOnDown    bool `gorm:"default:true"`
	NotifyOnUp      bool `gorm:"default:true"`
	NotifyAfterFail int  `gorm:"default:1"` // notify only after N consecutive failures

	Monitors []Monitor `gorm:"many2many:monitor_notifications"`
}

// ─── Integration ─────────────────────────────────────────────────────────────

// Integration stores external integration settings (one row per integration type).
type Integration struct {
	gorm.Model
	Name     string `gorm:"uniqueIndex;not null"` // e.g. "prometheus"
	Enabled  bool   `gorm:"default:false"`
	Config   string // JSON blob with integration-specific settings
}

// ─── Status Page ─────────────────────────────────────────────────────────────

type StatusPage struct {
	gorm.Model
	Name        string `gorm:"not null"`
	Slug        string `gorm:"uniqueIndex;not null"` // URL-friendly identifier
	Description string
	Public      bool `gorm:"default:false"`

	Monitors []Monitor `gorm:"many2many:status_page_monitors"`
	Users    []User    `gorm:"many2many:status_page_users"` // users with access
}
