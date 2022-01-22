package processor

import "github.com/kelseyhightower/envconfig"

type env struct {
	NumWorkers           int     `envconfig:"NUM_WORKERS" default:"300"`
	MaxWorkers           int     `envconfig:"MAX_WORKERS" default:"300"`
	FractionalNumWorkers float64 `envconfig:"FRACTIONAL_NUM_WORKERS" default:"0.25"`
	MaxAttempts          int     `envconfig:"MAX_ATTEMPTS" default:"3"`
	FixedAttempts        bool    `envconfig:"FIXED_ATTEMPTS" default:"true"`
	DelayAttempts        bool    `envconfig:"DELAY_ATTEMPTS" default:"false"`
	Timeout              int     `envconfig:"TIMEOUT" default:"-1"` // timeout in seconds, -1 is unlimited
}

var Config env

func init() {
	envconfig.MustProcess("", &Config)
}
