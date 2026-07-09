package config

// Application Constants
const (
	PortKey                  = "PORT"
	MaxConcurrentRequestsKey = "MAX_CONCURRENT_REQUESTS"
)

// Database Constants
const (
	ReadPGURIKey  = "READ_PG_URI"
	WritePGURIKey = "WRITE_PG_URI"
	usernameKey   = "DB_USERNAME"
	passwordKey   = "DB_PASSWORD"

	DBMaxOpenConnsKey           = "DB_MAX_OPEN_CONNS"
	DBMaxIdleConnsKey           = "DB_MAX_IDLE_CONNS"
	DBConnMaxLifetimeMinutesKey = "DB_CONN_MAX_LIFETIME_MINUTES"
	DBConnMaxIdleTimeMinutesKey = "DB_CONN_MAX_IDLE_TIME_MINUTES"
)
