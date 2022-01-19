package processor

import "time"

func QuadraticDelay(attempt int) time.Duration {
	return time.Duration(attempt * attempt * int(time.Second))
}
