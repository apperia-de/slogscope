package slogscope

type Config struct {
	LogLevel string    `yaml:"log_level"` // Global log level used as default.
	Packages []Package `yaml:"packages"`
}

type HandlerOptions struct {
	EnableFileWatcher bool
	ConfigFile        *string
	Config            *Config
	Debug             bool
}

type Package struct {
	Name     string `yaml:"name"`
	LogLevel string `yaml:"log_level"`
}
