package libstring

import (
	"net/http"
	"strings"
	"testing"
)

func TestStringInSlice(t *testing.T) {
	if StringInSlice([]string{"alice", "dan", "didip", "jason", "karl"}, "brotato") {
		t.Error("brotato should not be in slice.")
	}
}

func TestIPAddrFromRemoteAddr(t *testing.T) {
	if ipAddrFromRemoteAddr("127.0.0.1:8989") != "127.0.0.1" {
		t.Errorf("ipAddrFromRemoteAddr did not chop the port number correctly.")
	}
}

func TestRemoteIPDefault(t *testing.T) {
	ipLookups := []string{"RemoteAddr", "X-Real-IP"}
	ipv6 := "2601:7:1c82:4097:59a0:a80b:2841:b8c8"

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Real-IP", ipv6)

	ip := RemoteIP(ipLookups, request)
	if ip != request.RemoteAddr {
		t.Errorf("Did not get the right IP. IP: %v", ip)
	}
	if ip == ipv6 {
		t.Errorf("X-Real-IP should have been skipped. IP: %v", ip)
	}
}

func TestRemoteIPForwardedFor(t *testing.T) {
	ipLookups := []string{"X-Forwarded-For", "X-Real-IP", "RemoteAddr"}
	ipv6 := "2601:7:1c82:4097:59a0:a80b:2841:b8c8"

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Forwarded-For", "54.223.11.104")
	request.Header.Set("X-Real-IP", ipv6)

	ip := RemoteIP(ipLookups, request)
	if ip != "54.223.11.104" {
		t.Errorf("Did not get the right IP. IP: %v", ip)
	}
	if ip == ipv6 {
		t.Errorf("X-Real-IP should have been skipped. IP: %v", ip)
	}
}

func TestRemoteIPRealIP(t *testing.T) {
	ipLookups := []string{"X-Real-IP", "X-Forwarded-For", "RemoteAddr"}
	ipv6 := "2601:7:1c82:4097:59a0:a80b:2841:b8c8"

	request, err := http.NewRequest("GET", "/", strings.NewReader("Hello, world!"))
	if err != nil {
		t.Errorf("Unable to create new HTTP request. Error: %v", err)
	}

	request.Header.Set("X-Forwarded-For", "54.223.11.104")
	request.Header.Set("X-Real-IP", ipv6)

	ip := RemoteIP(ipLookups, request)
	if ip != ipv6 {
		t.Errorf("Did not get the right IP. IP: %v", ip)
	}
	if ip == "54.223.11.104" {
		t.Errorf("X-Forwarded-For should have been skipped. IP: %v", ip)
	}
}
