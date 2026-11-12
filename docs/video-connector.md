# Video Connector integration (PR #16)

The optional video bridge connects the Gateway to the existing **diagnostic** MCP Streamable HTTP server in Ploos AI Video Connector.

Configure the shared model registry with:

```json
{
  "video_connector": {
    "endpoint": "http://127.0.0.1:8787/mcp",
    "token_env": "PLOOS_VIDEO_CONNECTOR_TOKEN"
  }
}
```

Set a strong `PLOOS_VIDEO_CONNECTOR_TOKEN` in the Gateway environment and the same value as `PLOOS_MCP_BEARER_TOKEN` in the Video Connector environment. The endpoint must be a private/loopback IPv4 or IPv6 literal, with an explicit port and the exact `/mcp` path. No redirects or proxies are followed.

`GET /v1/video/capabilities` requires the Gateway's normal bearer token (`PLOOS_GATEWAY_API_KEY`). It calls the connector's MCP `video_capabilities` tool and returns its `result` JSON.

**No video generation is exposed by this PR.** The current connector returns diagnostic capability information only; it does not yet implement create/status/result operations. A future PR must implement and test those tools in the connector before the Gateway can expose corresponding job endpoints. This deliberately avoids a misleading or nonfunctional video generation API.
