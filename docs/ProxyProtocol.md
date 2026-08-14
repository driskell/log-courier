# PROXY Protocol Support

## Overview

Log Carver's TCP-based receivers (Courier and Stream) support the [PROXY protocol](https://www.haproxy.org/download/2.8/doc/proxy-protocol.txt), enabling them to sit behind a load balancer or TCP proxy (HAProxy, nginx, AWS NLB/CLB) while still seeing the real client address in logs, the admin API, and event metadata. Both the text-based v1 and binary v2 formats are accepted.

This is receiver-side only. Log Courier, as a client, is always the original source of a connection and has no upstream client to relay, so it has no PROXY protocol transports.

## Supported Transports

- **`tcp-proxy`** - Courier protocol over plain TCP, behind a PROXY protocol proxy
- **`tls-proxy`** - Courier protocol over TLS, behind a PROXY protocol proxy
- **`stream-proxy`** - Stream (line-based text) over plain TCP, behind a PROXY protocol proxy
- **`streamtls-proxy`** - Stream (line-based text) over TLS, behind a PROXY protocol proxy

Each is otherwise identical to its non-proxy counterpart (`tcp`, `tls`, `stream`, `streamtls`) - same options, same protocol on the wire after the header.

## Configuration Example

```yaml
receivers:
  - name: main
    listen:
      - 0.0.0.0:5000
    transport: tls-proxy
    ssl certificate: /etc/log-carver/server-cert.pem
    ssl key: /etc/log-carver/server-key.pem
    ssl client ca:
      - /etc/log-carver/client-ca.pem
    proxy protocol trusted sources:
      - 10.0.2.0/24
```

**Always set `proxy protocol trusted sources` unless the receiver is only reachable from the proxy already** (for example via network-level controls). A PROXY header lets the sender claim any source address it likes; without a trusted source list, any peer that can reach the port can present itself as any client IP. See [`proxy protocol trusted sources`](log-carver/Configuration.md#proxy-protocol-trusted-sources-receiver) in the Log Carver configuration reference.

You can check a configuration is valid before deploying it with:

```bash
$ ./log-carver -config /path/to/config.yaml -config-test
Configuration OK
```

## HAProxy Configuration

```
frontend courier_frontend
    bind *:5000
    mode tcp
    default_backend courier_backend

backend courier_backend
    mode tcp
    # send-proxy-v2 adds a v2 (binary) header; send-proxy adds a v1 (text) header
    server carver1 carver.example.com:5001 send-proxy-v2 check
    server carver2 carver.example.com:5002 send-proxy-v2 check backup
```

## Nginx Configuration

```nginx
stream {
    upstream courier_backend {
        server carver.example.com:5001;
        server carver.example.com:5002 backup;
    }

    server {
        listen 5000;
        proxy_pass courier_backend;
        proxy_protocol on;
    }
}
```

## How It Works

```
┌─────────┐         ┌───────┐            ┌────────┐
│ Client  │ ─TCP──→ │ Proxy │ ─PROXY+─→  │ Carver │
│ 1.2.3.4 │         │5.6.7.8│    TCP     │9.0.0.1 │
└─────────┘         └───────┘            └────────┘
                                Header: src=1.2.3.4
                                Sees: 1.2.3.4 (original client)
```

The PROXY header is the very first thing on the wire, ahead of anything else - including a TLS handshake. Log Carver reads it before anything else happens on the connection, then uses the original client address it carries for everything that would otherwise show the proxy's address: logs, the admin API, and event metadata.

A health check from the proxy itself (the `LOCAL` command) is accepted without needing a client address - the receiver simply reports the proxy's own address for those connections, as there is no client to report.

## Troubleshooting

### Connection is rejected immediately

- Check `proxy protocol trusted sources` includes the proxy's address, if set
- A `-proxy` transport requires a header on every connection - a direct connection without one (e.g. a health check probe that isn't the proxy) will be rejected

### Logs show the proxy's IP instead of the client's

- Confirm the receiver is using a `*-proxy` transport, not its plain counterpart
- Confirm the proxy is actually configured to send a header (`send-proxy`/`send-proxy-v2` in HAProxy, `proxy_protocol on` in nginx), not just forwarding traffic unchanged

### TLS handshake fails on a `tls-proxy`/`streamtls-proxy` receiver

- The PROXY header is read before the TLS handshake starts - if the proxy is configured to send a header only on some connections, the ones without it will fail the header requirement before TLS is ever attempted
- Otherwise, treat it as an ordinary TLS problem: check certificates and TLS version compatibility

## Migration Path

1. Deploy the receiver with a `-proxy` transport (existing plain/TLS receivers are unaffected and can run alongside it on a different port)
2. Deploy the proxy in front of it (HAProxy/nginx with PROXY protocol enabled)
3. Point clients at the proxy

No downtime required - the old and new receivers can coexist during the switch.
