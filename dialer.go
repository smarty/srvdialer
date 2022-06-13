package consuldialer

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
	return &simpleDialer{dialer: config.NetworkDialer, resolver: config.NameResolver}
}

type simpleDialer struct {
	dialer   NetworkDialer
	resolver NameResolver
}

func (this *simpleDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	if !isConsulService(address) {
		return this.dialer.DialContext(ctx, network, address)
	}

	_, records, err := net.DefaultResolver.LookupSRV(ctx, "", "", address)
	if err != nil {
		return nil, err
	} else if len(records) == 0 {
		return nil, &net.DNSError{Err: "NXDOMAIN", Name: address}
	}

	selected := records[0] // already sorted and randomized per RFC; for the moment, pick the top one rather than falling through each
	address = parseTargetAddress(selected.Target, selected.Port)
	return this.dialer.DialContext(ctx, network, address)
}
func isConsulService(value string) bool {
	return strings.HasSuffix(value, ".consul") && strings.Contains(value, ".service.")
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
