package consuldialer

import "net"

type configuration struct {
	NetworkDialer NetworkDialer
	NameResolver  NameResolver
}

func (singleton) NetworkDialer(value NetworkDialer) Option {
	return func(this *configuration) { this.NetworkDialer = value }
}
func (singleton) NameResolver(value NameResolver) Option {
	return func(this *configuration) { this.NameResolver = value }
}
func (singleton) apply(options ...Option) Option {
	return func(this *configuration) {
		for _, item := range Options.defaults(options...) {
			item(this)
		}
	}
}
func (singleton) defaults(options ...Option) []Option {
	return append([]Option{
		Options.NetworkDialer(&net.Dialer{}),
		Options.NameResolver(net.DefaultResolver),
	}, options...)
}

type singleton struct{}
type Option func(*configuration)

var Options singleton
