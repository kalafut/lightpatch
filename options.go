package lightpatch

import "time"

type config struct {
	noCRC   bool
	binary  bool
	timeout time.Duration
	base64  bool
}

type FuncOption func(*config)

func WithNoCRC() FuncOption {
	return func(o *config) {
		o.noCRC = true
	}
}

func WithBase64() FuncOption {
	return func(o *config) {
		o.base64 = true
	}
}
