package lightpatch

import "time"

type config struct {
	noCRC   bool
	binary  bool
	timeout time.Duration
}

type FuncOption func(*config)

func WithNoCRC() FuncOption {
	return func(o *config) {
		o.noCRC = true
	}
}

func WithBinary() FuncOption {
	return func(o *config) {
		o.binary = true
	}
}
