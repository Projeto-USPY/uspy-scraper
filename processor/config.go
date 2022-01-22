package processor

import "github.com/kelseyhightower/envconfig"

type env struct {
	NumWorkers           int     `envconfig:"NUM_WORKERS" default:"500"`
	MaxWorkers           int     `envconfig:"MAX_WORKERS" default:"300"`
	FractionalNumWorkers float64 `envconfig:"FRACTIONAL_NUM_WORKERS" default:"0.35"`
	MaxAttempts          int     `envconfig:"MAX_ATTEMPTS" default:"3"`
	FixedAttempts        bool    `envconfig:"FIXED_ATTEMPTS" default:"true"`
	DelayAttempts        bool    `envconfig:"DELAY_ATTEMPTS" default:"false"`
	Timeout              int     `envconfig:"TIMEOUT" default:"600"` // timeout in seconds
}

var Config env

func init() {
	envconfig.MustProcess("", &Config)
}
