package srvdialer

import (
	"context"
	"net"
)

type legacyDialer struct {
	ctx context.Context
	NetworkDialer
}

func NewLegacyDialer(ctx context.Context, dialer NetworkDialer) LegacyDialer {
	return &legacyDialer{ctx: ctx, NetworkDialer: dialer}
}

func (this *legacyDialer) Dial(network string, address string) (net.Conn, error) {
	return this.DialContext(this.ctx, network, address)
}
