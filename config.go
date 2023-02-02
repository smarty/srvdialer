package consuldialer

import (
	"fmt"
	"net"
	"time"
)

type configuration struct {
	NetworkDialer      NetworkDialer
	NameResolver       NameResolver
	InterceptedDomains []string
	InterceptedPorts   []int
	AllowedSuffixes    []string
}

func (singleton) NetworkDialer(value NetworkDialer) Option {
	return func(this *configuration) { this.NetworkDialer = value }
}
func (singleton) NameResolver(value NameResolver) Option {
	return func(this *configuration) { this.NameResolver = value }
}

// InterceptedDomains indicates what domain should be intercepted when performing a lookup. For example, if the
// intercepting domains are "k8s" and "consul" and the target address to be resolved is "mydomain.com", then standard
// domain-name resolution will occur. However, if the target address to be resolved is "something.consul", then the
// custom domain-name resolution will occur.
// The default intercepted domain is "consul".
func (singleton) InterceptedDomains(values ...string) Option {
	return func(this *configuration) {
		if len(values) == 0 {
			this.InterceptedDomains = this.InterceptedDomains[0:0]
		}

		this.InterceptedDomains = append(this.InterceptedDomains, values...)
	}
}

// InterceptedPorts indicates which port information will be intercepted if it is detected on the suffix of the target
// address to be resolved. For example, if [80, 443, 8080] are specified and the target address is "domain.consul:80",
// then the custom DNS-resolution logic will occur. (Note that at least one domain suffix in configured using
// InterceptedDomains must also match.) If no port information is detected, only domain-name based logic will occur.
// Lastly, if the port found on the request does not match one of the intercepted ports, then standard DNS resolution
// will occur. This interception logic is useful when relative to the built-in http.Client and http.Transport structs
// which append port information to each request's target address, e.g. http=>80, http=>443 such that "example.com"
// becomes "example.com:80". In such cases, this configuration allows the caller control over which endpoints get
// handled through the normal DNS resolution and which use the Consul-based DNS resolution.
// By default, port 80 and 443 are intercepted.
func (singleton) InterceptedPorts(values ...int) Option {
	return func(this *configuration) {
		if len(values) == 0 {
			this.InterceptedPorts = this.InterceptedPorts[0:0]
		}

		this.InterceptedPorts = append(this.InterceptedPorts, values...)
	}
}

func (singleton) apply(options ...Option) Option {
	return func(this *configuration) {
		for _, item := range Options.defaults(options...) {
			item(this)
		}

		for _, suffix := range this.InterceptedDomains {
			this.AllowedSuffixes = append(this.AllowedSuffixes, fmt.Sprintf(".%s", suffix))
			for _, port := range this.InterceptedPorts {
				this.AllowedSuffixes = append(this.AllowedSuffixes, fmt.Sprintf(".%s:%d", suffix, port))
			}
		}
	}
}
func (singleton) defaults(options ...Option) []Option {
	systemDialer := &net.Dialer{
		Timeout:   time.Second * 15,
		KeepAlive: time.Second * 30,
	}
	return append([]Option{
		Options.NetworkDialer(systemDialer),
		Options.NameResolver(net.DefaultResolver),
		Options.InterceptedDomains("consul"),
		Options.InterceptedPorts(80, 443),
	}, options...)
}

type singleton struct{}
type Option func(*configuration)

var Options singleton
