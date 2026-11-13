package srvdialer

import (
	"context"
	"errors"
	"net"
	"reflect"
	"syscall"
	"testing"
)

func TestDialFallsThroughToNextRecordOnFailure(t *testing.T) {
	dialer := &fakeDialer{errors: map[string]error{"10.0.0.1:80": refused(), "10.0.0.2:80": refused()}}
	resolver := &fakeResolver{records: []*net.SRV{
		{Target: "10.0.0.1", Port: 80},
		{Target: "10.0.0.2", Port: 80},
		{Target: "10.0.0.3", Port: 80},
	}}
	subject := New(Options.NetworkDialer(dialer), Options.NameResolver(resolver))

	conn, err := subject.DialContext(context.Background(), "tcp", "street.service.consul:80")

	if err != nil || conn == nil {
		t.Fatalf("expected a connection, got conn=%v err=%v", conn, err)
	}
	assertDialed(t, dialer, "10.0.0.1:80", "10.0.0.2:80", "10.0.0.3:80")
}

func TestDialStopsAtFirstSuccess(t *testing.T) {
	dialer := &fakeDialer{}
	resolver := &fakeResolver{records: []*net.SRV{{Target: "10.0.0.1", Port: 80}, {Target: "10.0.0.2", Port: 80}}}
	subject := New(Options.NetworkDialer(dialer), Options.NameResolver(resolver))

	_, err := subject.DialContext(context.Background(), "tcp", "street.service.consul:80")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDialed(t, dialer, "10.0.0.1:80")
}

func TestDialReturnsLastErrorWhenAllRecordsFail(t *testing.T) {
	last := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.EHOSTUNREACH}
	dialer := &fakeDialer{errors: map[string]error{"10.0.0.1:80": refused(), "10.0.0.2:80": last}}
	resolver := &fakeResolver{records: []*net.SRV{{Target: "10.0.0.1", Port: 80}, {Target: "10.0.0.2", Port: 80}}}
	subject := New(Options.NetworkDialer(dialer), Options.NameResolver(resolver))

	conn, err := subject.DialContext(context.Background(), "tcp", "street.service.consul:80")

	if conn != nil || !errors.Is(err, syscall.EHOSTUNREACH) {
		t.Fatalf("expected the last dial error, got conn=%v err=%v", conn, err)
	}
	assertDialed(t, dialer, "10.0.0.1:80", "10.0.0.2:80")
}

func TestDialStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dialer := &fakeDialer{errors: map[string]error{"10.0.0.1:80": context.Canceled}, onDial: cancel}
	resolver := &fakeResolver{records: []*net.SRV{{Target: "10.0.0.1", Port: 80}, {Target: "10.0.0.2", Port: 80}}}
	subject := New(Options.NetworkDialer(dialer), Options.NameResolver(resolver))

	_, err := subject.DialContext(ctx, "tcp", "street.service.consul:80")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	assertDialed(t, dialer, "10.0.0.1:80")
}

func TestNonServiceAddressIsDialedDirectly(t *testing.T) {
	dialer := &fakeDialer{}
	subject := New(Options.NetworkDialer(dialer), Options.NameResolver(&fakeResolver{}))

	_, _ = subject.DialContext(context.Background(), "tcp", "example.com:80")

	assertDialed(t, dialer, "example.com:80")
}

func TestParseTargetAddress(t *testing.T) {
	cases := map[string]string{
		parseTargetAddress("0a000001.addr.dc1.consul.", 8080): "10.0.0.1:8080",
		parseTargetAddress("host.node.consul.", 80):           "host.node.consul.:80",
		parseTargetAddress("host.node.consul.", 0):            "host.node.consul.",
	}
	for actual, expected := range cases {
		if actual != expected {
			t.Errorf("expected %q, got %q", expected, actual)
		}
	}
}

func refused() error { return &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED} }

func assertDialed(t *testing.T, dialer *fakeDialer, expected ...string) {
	t.Helper()
	if !reflect.DeepEqual(dialer.dialed, expected) {
		t.Fatalf("expected dials %v, got %v", expected, dialer.dialed)
	}
}

type fakeDialer struct {
	errors map[string]error
	dialed []string
	onDial func()
}

func (this *fakeDialer) DialContext(_ context.Context, _ string, address string) (net.Conn, error) {
	this.dialed = append(this.dialed, address)
	if this.onDial != nil {
		this.onDial()
	}
	if err := this.errors[address]; err != nil {
		return nil, err
	}
	client, server := net.Pipe()
	_ = server.Close()
	return client, nil
}

type fakeResolver struct{ records []*net.SRV }

func (this *fakeResolver) LookupSRV(context.Context, string, string, string) (string, []*net.SRV, error) {
	return "", this.records, nil
}
