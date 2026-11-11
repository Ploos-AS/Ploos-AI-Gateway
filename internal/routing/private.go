package routing

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
)

// validatePrivateEndpoint accepts only literal private or loopback IP endpoints.
func validatePrivateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return errors.New("invalid private endpoint")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("private endpoint requires HTTP or HTTPS")
	}
	if u.Path != "/v1/chat/completions" {
		return errors.New("private endpoint requires /v1/chat/completions")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || port == "" {
		return errors.New("private endpoint requires an IP and explicit port")
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || ip.Zone() != "" || ip.Is4In6() || (!ip.IsPrivate() && !ip.IsLoopback()) {
		return errors.New("private endpoint requires a private or loopback IP literal")
	}
	if _, err := net.LookupPort("tcp", port); err != nil {
		return errors.New("invalid private endpoint port")
	}
	return nil
}
