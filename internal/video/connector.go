package video

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Endpoint string `json:"endpoint"`
	TokenEnv string `json:"token_env"`
}

type Connector struct {
	endpoint string
	token    string
	client   *http.Client
}

func httpRedirectPolicy(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

func New(cfg Config) (*Connector, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/mcp" {
		return nil, errors.New("video connector requires a private HTTP /mcp endpoint")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || port == "" {
		return nil, errors.New("video connector requires an IP and port")
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || ip.Zone() != "" || ip.Is4In6() || (!ip.IsPrivate() && !ip.IsLoopback()) {
		return nil, errors.New("video connector requires a private IP literal")
	}
	if _, err := net.LookupPort("tcp", port); err != nil {
		return nil, errors.New("invalid video connector port")
	}
	if cfg.TokenEnv != "PLOOS_VIDEO_CONNECTOR_TOKEN" {
		return nil, errors.New("video connector token environment must be PLOOS_VIDEO_CONNECTOR_TOKEN")
	}
	token := os.Getenv(cfg.TokenEnv)
	if len(token) < 32 || strings.TrimSpace(token) != token {
		return nil, errors.New("missing or invalid video connector token")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	client := &http.Client{}
	client.Timeout = 10 * time.Second
	client.Transport = transport
	client.CheckRedirect = httpRedirectPolicy
	return &Connector{endpoint: cfg.Endpoint, token: token, client: client}, nil
}

// Capabilities queries only the connector's diagnostic MCP capability tool.
// No generation requests are submitted by this bridge.
func (c *Connector) Capabilities(ctx context.Context) (json.RawMessage, error) {
	payload := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"video_capabilities","arguments":{}}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.New("video connector unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return nil, errors.New("video connector rejected capability request")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 65537))
	if err != nil || len(data) > 65536 {
		return nil, errors.New("invalid video connector response")
	}
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.JSONRPC != "2.0" || len(envelope.Result) == 0 || len(envelope.Error) != 0 {
		return nil, errors.New("invalid video connector RPC response")
	}
	return envelope.Result, nil
}
