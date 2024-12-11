package zookeeper

import (
	"time"
)

// nolint
type options struct {
	Client *Client
}

// Option will define a function of handling Options
type Option func(*options)

type zkClientOption func(*Client)

// WithZkTimeOut sets zk Client timeout
func WithZkTimeOut(t time.Duration) zkClientOption {
	return func(opt *Client) {
		opt.timeout = t
	}
}
