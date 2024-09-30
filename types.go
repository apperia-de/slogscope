package slogscope

type Config struct {
	LogLevel string    `yaml:"log_level"` // Global log level used as default.
	Packages []Package `yaml:"packages"`
}

type HandlerOptions struct {
	Debug             bool
	Config            *Config
	ConfigFile        *string
	EnableFileWatcher bool
}

type Package struct {
	Name     string `yaml:"name"`
	LogLevel string `yaml:"log_level"`
}
