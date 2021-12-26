package config

// General structure of config file
type General struct {
	Info Info `json:"info,omitempty"`
}

// Info represents general information schema
type Info struct {
	Endpoint string `json:"endpoint,omitempty"`
	Port     int    `json:"port,omitempty"`
	LogPath  string `json:"logPath,omitempty"`
	LogLevel string `json:"logLevel,omitempty"`
}
