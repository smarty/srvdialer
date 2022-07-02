package consuldialer

import (
	"context"
	"net"
)

type LegacyDialer interface {
	Dial(string, string) (net.Conn, error)
	NetworkDialer
}

type NetworkDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}
type NameResolver interface {
	LookupSRV(context.Context, string, string, string) (string, []*net.SRV, error)
}
