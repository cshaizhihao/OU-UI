# Hysteria2 Provider

This document describes the OU-UI v0.6.0 Hysteria2 provider. It builds and validates a server configuration model, renders Hysteria2 YAML, writes it as an active runtime config, and generates an OU-UI managed per-node systemd service. It does not download packages, open firewall rules, or request certificates.

The managed service uses:

```text
hysteria server -c <configPath>
```

`configPath`, `configDir`, `unitPath`, `serviceMode`, and `managedByOuui` are returned in `node.deploy` task results and mirrored into Node state.

## Input model

The provider consumes `provider.NodeSpec`:

- `runtime`: must be empty or `hysteria2`.
- `listen`: Hysteria2 listen address such as `:443`, `0.0.0.0:443`, or `[::]:443`.
- `port`: optional fallback used when `listen` is empty.
- `settings.tls.certPath`: TLS certificate path.
- `settings.tls.keyPath`: TLS private key path.
- `settings.auth.password`: Hysteria2 password authentication secret supplied by the caller.
- `settings.bandwidth.up`: optional server upload limit, for example `100 mbps`.
- `settings.bandwidth.down`: optional server download limit, for example `100 mbps`.
- `settings.bandwidth.ignoreClientBandwidth`: optional Hysteria2 `ignoreClientBandwidth` flag.
- `settings.masquerade`: optional Hysteria2 masquerade config.
- `settings.userTrafficLimit`: reserved OU-UI model for future per-user traffic limits.

Compatibility aliases are accepted for early UI experiments:

- `settings.tlsCertPath` and `settings.tlsKeyPath`
- `settings.authPassword`
- `settings.password`
- `settings.trafficLimit`

## Validation

`Validate` checks the preview contract only:

- listen port is required, either from `listen` or `port`;
- listen port must be numeric and between `1` and `65535`;
- listen port ranges and `realm://` listen URIs are not supported by the current provider;
- TLS certificate path is required;
- TLS key path is required;
- auth password is required;
- masquerade requires a supported `type`: `file`, `proxy`, or `string`;
- `file` masquerade requires `file.dir`;
- `proxy` masquerade requires an absolute `proxy.url`;
- `string` masquerade requires a `string` block.

Validation does not check whether certificate files exist before deployment. The generated managed service will fail health if systemd cannot keep the runtime active.

## Rendered YAML

Minimal example:

```yaml
listen: ":443"
tls:
  cert: "/etc/ou-ui/certs/fullchain.pem"
  key: "/etc/ou-ui/certs/privkey.pem"
auth:
  type: "password"
  password: "replace-with-runtime-secret"
```

With bandwidth and proxy masquerade:

```yaml
listen: "0.0.0.0:443"
tls:
  cert: "/etc/ou-ui/certs/fullchain.pem"
  key: "/etc/ou-ui/certs/privkey.pem"
auth:
  type: "password"
  password: "replace-with-runtime-secret"
bandwidth:
  up: "100 mbps"
  down: "100 mbps"
masquerade:
  type: "proxy"
  proxy:
    url: "https://example.com/"
    rewriteHost: true
    insecure: false
    xForwarded: false
```

## Traffic limit reservation

`settings.userTrafficLimit` is intentionally kept out of the rendered Hysteria2 YAML. It exists so the panel and task payloads can converge on a stable field name before OU-UI adds a runtime accounting backend or Hysteria2 auth integration that can enforce per-user quotas.

Do not store real passwords, certificate private keys, tokens, or provider API credentials in repository files. Values in examples must remain placeholders.
