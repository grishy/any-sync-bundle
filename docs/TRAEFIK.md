# Using Traefik Proxy with Any-Sync-Bundle

This guide explains how to set up Traefik as a reverse proxy for any-sync-bundle.  
**Important:** This setup is different from typical HTTP reverse proxying because any-sync-bundle uses custom protocols (DRPC over TCP and QUIC).

## Overview

Any-sync-bundle uses two network protocols:

- **TCP port 33010** - DRPC protocol for coordinator, consensus, filenode, and sync services
- **UDP port 33020** - QUIC protocol (custom, not HTTP/3) for faster transport

Unlike HTTP services, these are custom binary protocols. Traefik can forward them using **TCP and UDP routers**, but advanced features like path-based routing, header manipulation, or SNI-based multi-backend routing won't work.

### What Traefik Can Do

✅ Forward TCP port 33010 to any-sync-bundle
✅ Forward UDP port 33020 to any-sync-bundle
✅ Handle one backend instance per Traefik deployment
✅ Provide load balancing across multiple any-sync-bundle instances (simple round-robin)

### What Traefik Cannot Do

❌ Route different domains to different backends (no HTTP headers to inspect)
❌ TLS termination (any-sync-bundle doesn't use TLS)
❌ Path-based routing (no HTTP paths)
❌ SNI-based routing (any-sync-bundle doesn't use TLS with SNI)

## Prerequisites

- **Traefik v2.9+** or **v3.x** (this guide uses v3.x syntax)
- **Docker & Docker Compose**
- Basic understanding of Traefik concepts (EntryPoints, Routers, Services)
- Firewall configured to allow TCP 33010 and UDP 33020

### 8. Test Connectivity

From another machine, test if ports are reachable:

```bash
# Test TCP
nc -zv your-server.example.com 33010

# Test UDP (requires netcat with UDP support)
nc -zvu your-server.example.com 33020
```

## Troubleshooting

### UDP Port Not Working

**Symptom:** Clients connect but sync is slow or fails intermittently

**Causes:**

1. **Firewall blocking UDP** - Most common issue

   ```bash
   # Check firewall rules
   sudo ufw status
   sudo iptables -L -n | grep 33020
   ```

2. **NAT/Router not forwarding UDP**

   - Configure port forwarding on your router: UDP 33020 → your server
   - Some ISPs block UDP - test with TCP only first

3. **Docker not binding UDP correctly**
   ```bash
   # Verify Docker is listening
   sudo netstat -ulnp | grep 33020
   ```

**Solution:**

- Ensure UFW/iptables allows UDP 33020
- Test with a simpler UDP echo server to verify network path
- Check Docker logs: `docker logs traefik`

### Client Cannot Connect

**Symptom:** Client config loaded but connection fails

**Diagnosis:**

1. Check `client-config.yml` addresses:

   ```bash
   cat ./data/client-config.yml
   ```

2. Verify addresses match your public IP/hostname

3. Test from client machine:
   ```bash
   telnet your-server.example.com 33010
   ```

**Solutions:**

- Wrong external address → Edit `./data/bundle-config.yml`, change `externalAddr:` list, restart
- Firewall → Check TCP 33010 and UDP 33020
- DNS → Ensure hostname resolves correctly from client

## Advanced Topics

### PROXY Protocol

If you need to preserve client IP addresses, Traefik supports PROXY protocol for TCP:

```yaml
# In Traefik command
- "--entrypoints.any-sync-tcp.proxyProtocol.trustedIPs=192.168.1.0/24"

# In any-sync-bundle labels
- "traefik.tcp.services.any-sync-tcp-service.loadbalancer.proxyProtocol.version=2"
```

**Note:** This requires any-sync-bundle to support PROXY protocol, which it currently does not. This is for future reference.

## Alternatives

Traefik might not be the best fit for all use cases. Consider alternatives:

### Direct Exposure

**Best for:** Simple deployments, single server, no need for routing logic

```yaml
services:
  any-sync-bundle:
    ports:
      - "33010:33010"
      - "33020:33020/udp"
```

**Pros:** Simple, no extra layer, better performance
**Cons:** No reverse proxy features, less flexible

### HAProxy

**Best for:** Pure TCP/UDP forwarding with advanced health checks

HAProxy has better TCP/UDP handling and health check options.

### Nginx Stream Module

**Best for:** Simple TCP/UDP proxying without dynamic configuration needs

```nginx
stream {
    upstream any_sync_tcp {
        server any-sync-bundle:33010;
    }

    server {
        listen 33010;
        proxy_pass any_sync_tcp;
    }
}
```

For questions or issues, please open an issue on the [GitHub repository](https://github.com/grishy/any-sync-bundle/issues).
