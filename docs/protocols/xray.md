# Xray Provider Protocol

OU-UI v0.6.0 validates node input, renders an Xray JSON configuration, writes it as an active runtime config, and generates an OU-UI managed per-node systemd service. It still does not download Xray binaries, open firewall rules, or run package-manager commands.

The managed service uses:

```text
xray run -config <configPath>
```

`configPath`, `configDir`, `unitPath`, `serviceMode`, and `managedByOuui` are returned in `node.deploy` task results and mirrored into Node state.

## NodeSpec Shape

The provider accepts `internal/provider.NodeSpec`:

```json
{
  "runtime": "xray",
  "protocol": "vless",
  "listen": "0.0.0.0",
  "port": 443,
  "settings": {}
}
```

Common fields:

| Field | Required | Notes |
| --- | --- | --- |
| `runtime` | yes | Must be `xray` for dispatchers that check runtime. |
| `protocol` | yes | One of `vless`, `vmess`, `trojan`, `shadowsocks`. |
| `listen` | no | Valid IP address. Defaults to `0.0.0.0` when omitted. |
| `port` | yes | Integer from `1` to `65535`. |
| `settings.id` | no | Used as rendered inbound tag input and as UUID fallback. |
| `settings.remark` | no | Stored in the typed model for UI/display use; not rendered to Xray JSON. |

## Protocol Settings

### VLESS

Required:

| Setting | Notes |
| --- | --- |
| `uuid` | Client UUID. `id` is accepted as a fallback. |

Optional:

| Setting | Default |
| --- | --- |
| `flow` | empty |
| `encryption` | `none` |

### VMess

Required:

| Setting | Notes |
| --- | --- |
| `uuid` | Client UUID. `id` is accepted as a fallback. |

Optional:

| Setting | Default |
| --- | --- |
| `alterId` | `0` |
| `security` | `auto` |

### Trojan

Required:

| Setting | Notes |
| --- | --- |
| `password` | Client password. `uuid` and `id` are accepted as fallbacks. |

### Shadowsocks

Required:

| Setting | Notes |
| --- | --- |
| `password` | Server password. |

Optional:

| Setting | Default |
| --- | --- |
| `method` | `aes-128-gcm` |
| `network` | `tcp,udp` |

## Reality

Reality is supported only with VLESS.

Enable Reality by setting any of:

| Setting | Value |
| --- | --- |
| `reality` | `true` |
| `reality.enabled` | `true` |
| `realityEnabled` | `true` |
| `reality.dest` | non-empty destination |

Required when enabled:

| Setting | Notes |
| --- | --- |
| `reality.dest` or `dest` | Destination host and port, for example `www.example.com:443`. |
| `reality.privateKey` or `privateKey` | Xray Reality private key. |
| `reality.serverNames`, `reality.serverName`, `serverNames`, or `serverName` | At least one server name. Comma-separated string and string arrays are accepted. |

Optional:

| Setting | Default |
| --- | --- |
| `reality.publicKey` or `publicKey` | Empty. Useful for panel display, not rendered into inbound server config. |
| `reality.shortIds`, `reality.shortId`, `shortIds`, or `shortId` | empty array |
| `reality.spiderX` or `spiderX` | `/` |

## Rendered Xray JSON

`Render` returns indented JSON with:

- one inbound for the requested protocol;
- protocol-specific `settings`;
- Reality `streamSettings` when enabled;
- sniffing enabled for `http`, `tls`, and `quic`;
- `freedom` and `blackhole` outbounds.

Example VLESS + Reality input:

```json
{
  "runtime": "xray",
  "protocol": "vless",
  "listen": "0.0.0.0",
  "port": 443,
  "settings": {
    "id": "node-443",
    "uuid": "00000000-0000-0000-0000-000000000000",
    "flow": "xtls-rprx-vision",
    "reality": true,
    "reality.dest": "www.example.com:443",
    "reality.serverNames": ["www.example.com"],
    "reality.privateKey": "REPLACE_WITH_GENERATED_PRIVATE_KEY",
    "reality.shortIds": ["0123456789abcdef"]
  }
}
```

The provider treats key material as opaque strings. Tests and examples must use placeholders only; do not commit real tokens, private keys, or production passwords.
