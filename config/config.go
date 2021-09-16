package config

type Config struct {
	Endpoint     string // CSI endpoint
	NodeID       string // CSI node ID
	HeketiURL    string // Heketi endpoint
	HeketiUser   string // Heketi user name who has access to create and delete volume
	HeketiSecret string // Heketi user password
}

//NewConfig returns config struct to initialize new driver
func NewConfig() *Config {
	return &Config{}
}
