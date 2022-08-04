package releaser

type Configuration struct {
	SrcCfg []SrcConfiguration `mapstructure:"sources"`
	DstCfg DstConfiguration   `mapstructure:"destination"`
}

type DstConfiguration struct {
	Owner string `mapstructure:"owner"`
	Repo  string `mapstructure:"repo"`
}

type SrcConfiguration struct {
	Name    string   `mapstructure:"name"`
	Version string   `mapstructure:"version"`
	Repo    string   `mapstructure:"repo"`
	Charts  []string `mapstructure:"charts"`
}
