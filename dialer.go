package srvdialer

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
)

func New(options ...Option) NetworkDialer {
	var config configuration
	Options.apply(options...)(&config)
	return &simpleDialer{dialer: config.NetworkDialer, resolver: config.NameResolver, allowedSuffixes: config.AllowedSuffixes}
}

type simpleDialer struct {
	dialer          NetworkDialer
	resolver        NameResolver
	allowedSuffixes []string
}

func (this *simpleDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	if !this.isService(address) {
		return this.dialer.DialContext(ctx, network, address)
	}

	if index := strings.LastIndex(address, ":"); index > 0 {
		address = address[0:index] // remove the port information
	}

	_, records, err := this.resolver.LookupSRV(ctx, "", "", address)
	if err != nil {
		return nil, err
	} else if len(records) == 0 {
		return nil, &net.DNSError{Err: "NXDOMAIN", Name: address}
	}

	// Records are already sorted by priority and randomized by weight per RFC 2782. Try each in that order: a failed
	// connect never reached the target, so moving on to the next one is always safe. Stop once the caller's context
	// ends, since every remaining attempt would fail the same way.
	for _, selected := range records {
		var conn net.Conn
		if conn, err = this.dialer.DialContext(ctx, network, parseTargetAddress(selected.Target, selected.Port)); err == nil {
			return conn, nil
		} else if ctx.Err() != nil {
			break
		}
	}

	return nil, err // the last attempt's error, so callers can still inspect it with errors.Is/As
}
func (this *simpleDialer) isService(value string) bool {
	return this.containsAllowedSuffix(value) && strings.Contains(value, ".service.")
}
func (this *simpleDialer) containsAllowedSuffix(value string) bool {
	for _, suffix := range this.allowedSuffixes {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}

	return false
}
func parseTargetAddress(address string, port uint16) string {
	if port == 0 {
		return address
	}

	index := strings.Index(address, ".addr.")
	if index < 0 {
		return fmt.Sprintf("%s:%d", address, port)
	}

	rawIPv4 := address[0:index]
	binaryIPv4, err := hex.DecodeString(rawIPv4)
	if err != nil || len(binaryIPv4) < 4 {
		return fmt.Sprintf("%s:%d", address, port)
	}

	return fmt.Sprintf("%d.%d.%d.%d:%d", binaryIPv4[0], binaryIPv4[1], binaryIPv4[2], binaryIPv4[3], port)
}
