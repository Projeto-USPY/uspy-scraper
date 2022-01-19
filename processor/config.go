package processor

import "github.com/kelseyhightower/envconfig"

type env struct {
	Processor struct {
		NumWorkers  int `envconfig:"NUM_WORKERS" default:"50"`
		MaxAttempts int `envconfig:"MAX_ATTEMPTS" default:"5"`
		Timeout     int `envconfig:"TIMEOUT" default:"300"` // timeout in seconds
	}
}

var Config env

func init() {
	envconfig.MustProcess("", &Config)
}
