# SSH Certificate Authority

MageBox Team Server uses SSH Certificate Authority (CA) for secure, scalable access management. This eliminates the need to deploy individual SSH keys to each server.

## How It Works

### Traditional SSH Keys (The Problem)

```
┌─────────────┐     Deploy key to     ┌─────────────┐
│    Alice    │ ───────────────────▶  │  Server 1   │
│  (new user) │                       │  Server 2   │
│             │  Must update ALL      │  Server 3   │
│             │  servers when Alice   │  ...        │
│             │  joins or leaves      │  Server N   │
└─────────────┘                       └─────────────┘

Problems:
- Every user change = update every server
- Forgotten key removal = security breach
- No automatic expiration
- Scales poorly: 100 users × 50 servers = 5000 entries
```

### SSH CA (The Solution)

```
┌─────────────────────────────────────────────────────────────┐
│                      TEAM SERVER                            │
│                                                             │
│    ┌──────────────────────────────────────────────────┐    │
│    │            CA Private Key                         │    │
│    │     (signs certificates, never leaves server)     │    │
│    └──────────────────────────────────────────────────┘    │
│                          │                                  │
│                          │ Signs user certificates          │
│                          ▼                                  │
└─────────────────────────────────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
    ┌───────────┐    ┌───────────┐    ┌───────────┐
    │ Server 1  │    │ Server 2  │    │ Server 3  │
    │           │    │           │    │           │
    │ CA Public │    │ CA Public │    │ CA Public │
    │ Key ONLY  │    │ Key ONLY  │    │ Key ONLY  │
    │           │    │           │    │           │
    │ (deployed │    │ (deployed │    │ (deployed │
    │  once)    │    │  once)    │    │  once)    │
    └───────────┘    └───────────┘    └───────────┘

Benefits:
- User changes = NO server updates needed
- Certificates auto-expire (default: 24 hours)
- Revocation is instant (stop issuing new certs)
- Scales perfectly: 1 CA key per server, regardless of users
```

## Architecture

### Components

| Component | Location | Purpose |
|-----------|----------|---------|
| CA Private Key | Team Server (encrypted) | Signs user certificates |
| CA Public Key | Each target server | Validates certificates |
| User Private Key | User's machine (`~/.magebox/keys/`) | Proves identity |
| User Certificate | User's machine (`~/.magebox/keys/`) | Grants access (time-limited) |

### Trust Chain

```
CA Private Key (Team Server)
        │
        │ signs
        ▼
User Certificate ──────────▶ Target Server
        │                          │
        │                          │ validates against
        │                          ▼
        │                    CA Public Key
        │
        └── contains ──▶ User's Public Key
                         Validity Period (24h)
                         Allowed Principals (usernames)
                         Extensions (pty, port-forward, etc.)
```

## Lifecycle

### 1. Server Initialization

```bash
magebox server init --data-dir /var/lib/magebox/teamserver
```

Output:
```
MageBox Team Server initialized successfully!

Master Key (save securely):
  abc123def456...

Admin Token:
  token_xyz789...

CA Public Key (for target servers):
  ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... magebox-ca

The CA public key will be automatically deployed when you add environments.
```

The CA key pair is generated automatically:
- **CA Private Key**: Stored encrypted in Team Server database
- **CA Public Key**: Displayed and stored for deployment

### 2. Add Environment

```bash
magebox server env add production \
    --project myapp \
    --host prod.example.com \
    --port 22 \
    --deploy-user deploy \
    --deploy-key ~/.ssh/deploy_key
```

What happens on the target server:

1. CA public key copied to `/etc/ssh/magebox-ca.pub`
2. `sshd_config` updated with:
   ```
   TrustedUserCAKeys /etc/ssh/magebox-ca.pub
   ```
3. SSH daemon reloaded

::: tip One-Time Setup
The CA public key only needs to be deployed once per server. Adding or removing users requires NO changes to target servers.
:::

### 3. User Joins

```bash
magebox server join https://team.example.com --token INVITE_TOKEN
```

What happens:

1. Team Server generates Ed25519 key pair for user
2. Team Server signs the public key with CA, creating a certificate
3. User receives:
   - Private key → `~/.magebox/keys/teamserver_key`
   - Certificate → `~/.magebox/keys/teamserver_key-cert.pub`

Certificate contents:
```
Type: ssh-ed25519-cert-v01@openssh.com user certificate
Key ID: "alice@example.com"
Serial: 1705312800
Valid: from 2025-01-15T09:00:00 to 2025-01-16T09:00:00
Principals:
        deploy
Extensions:
        permit-pty
        permit-user-rc
```

### 4. SSH Connection

```bash
magebox ssh myapp/production
```

Connection flow:

```
┌──────────────┐                        ┌──────────────┐
│    User      │                        │   Server     │
│              │                        │              │
│ 1. Present   │ ────────────────────▶  │ 2. Verify    │
│    cert +    │                        │    - Signed  │
│    prove     │                        │      by CA?  │
│    key       │                        │    - Not     │
│    ownership │                        │      expired?│
│              │                        │    - Valid   │
│              │                        │      principal│
│              │  ◀────────────────────  │              │
│              │     3. Access granted  │              │
└──────────────┘                        └──────────────┘
```

### 5. Certificate Renewal

Certificates expire after 24 hours (configurable). Users renew with:

```bash
magebox cert renew
```

What happens:

1. Client sends renewal request with session token
2. Team Server verifies:
   - User is still active
   - User still has project access
3. Team Server issues new certificate (24h validity)
4. Old certificate naturally expires

::: warning Automatic Renewal
Consider setting up a cron job or shell alias to auto-renew:
```bash
# Add to ~/.bashrc or ~/.zshrc
alias mssh='magebox cert renew --quiet && magebox ssh'
```
:::

### 6. Access Revocation

```bash
magebox server user revoke alice --project myapp
```

What happens:

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  Team Server marks access as revoked                        │
│                                                             │
│  Alice's current certificate:                               │
│  ┌─────────────────────────────────────────────┐           │
│  │ Still valid for up to 24 hours              │           │
│  │ (max exposure window)                       │           │
│  └─────────────────────────────────────────────┘           │
│                                                             │
│  When Alice tries to renew:                                 │
│  ┌─────────────────────────────────────────────┐           │
│  │ ✗ Error: Access revoked, cannot renew       │           │
│  └─────────────────────────────────────────────┘           │
│                                                             │
│  Result:                                                    │
│  - No changes needed on ANY server                          │
│  - Access automatically expires within 24h                  │
│  - Immediate for any new connection attempts after expiry   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

::: tip Shorter Validity for Sensitive Environments
For high-security environments, use shorter certificate validity:
```bash
magebox server start --cert-validity 1h
```
Shorter validity = faster revocation, but more frequent renewals.
:::

## Configuration

### Server Options

| Option | Default | Description |
|--------|---------|-------------|
| `--cert-validity` | `24h` | Certificate validity duration |
| `--cert-principals` | `deploy` | Default principals (login usernames) |
| `--cert-extensions` | `permit-pty` | SSH extensions to allow |

Environment variables:
```bash
export MAGEBOX_CERT_VALIDITY=24h
export MAGEBOX_CERT_PRINCIPALS=deploy,ubuntu
export MAGEBOX_CERT_EXTENSIONS=permit-pty,permit-port-forwarding
```

### Certificate Validity Options

| Duration | Use Case |
|----------|----------|
| `1h` | High-security, frequent access |
| `8h` | Standard workday |
| `24h` | Default, good balance |
| `168h` | Weekly renewal (lower security) |

## CLI Reference

### Certificate Commands

```bash
# Renew certificate (get new 24h cert)
magebox cert renew

# Renew quietly (for scripts)
magebox cert renew --quiet

# Show certificate info
magebox cert show

# Show certificate expiry
magebox cert expiry
```

### Server Commands

```bash
# Show CA public key
magebox server ca show

# Export CA public key (for manual deployment)
magebox server ca export > magebox-ca.pub

# Rotate CA key (advanced, requires re-deployment)
magebox server ca rotate
```

## API Reference

### Certificate Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/cert/renew` | POST | Session | Renew user certificate |
| `/api/cert/info` | GET | Session | Get certificate details |

### Renewal Request

```bash
curl -X POST \
     -H "Authorization: Bearer SESSION_TOKEN" \
     https://team.example.com/api/cert/renew
```

Response:
```json
{
  "certificate": "ssh-ed25519-cert-v01@openssh.com AAAAI...",
  "valid_until": "2025-01-16T09:00:00Z",
  "principals": ["deploy"]
}
```

## Security Considerations

### CA Key Protection

The CA private key is the most critical secret:

- Stored encrypted (AES-256-GCM) in Team Server database
- Never transmitted over network
- Never written to disk unencrypted
- Compromise = must rotate CA and re-deploy to all servers

::: danger CA Key Compromise
If you suspect CA key compromise:
1. Immediately rotate: `magebox server ca rotate`
2. Re-deploy CA public key to all servers
3. All users must re-join to get new certificates
:::

### Certificate Security

| Attack Vector | Mitigation |
|---------------|------------|
| Stolen certificate | Auto-expires in 24h |
| Stolen private key | Revoke user, key becomes useless after cert expiry |
| Man-in-the-middle | Certificate binds key to specific user identity |
| Replay attacks | Certificates include timestamp, checked by sshd |

### Audit Trail

All certificate operations are logged:

```bash
magebox server audit --action CERT_ISSUE
magebox server audit --action CERT_RENEW
magebox server audit --action CERT_DENY
```

Logged fields:
- User identity
- Timestamp
- Certificate serial number
- Validity period
- Client IP address
- Success/failure reason

## Manual CA Deployment

If automatic deployment fails, manually configure target servers:

### 1. Get CA Public Key

```bash
magebox server ca export > magebox-ca.pub
```

### 2. Copy to Target Server

```bash
scp magebox-ca.pub root@server:/etc/ssh/magebox-ca.pub
```

### 3. Configure SSHD

Add to `/etc/ssh/sshd_config`:
```
TrustedUserCAKeys /etc/ssh/magebox-ca.pub
```

### 4. Reload SSH

```bash
sudo systemctl reload sshd
```

## Comparison with Other Methods

| Method | Key Management | Revocation | Audit | Scalability |
|--------|---------------|------------|-------|-------------|
| **SSH CA** | Centralized | Instant | Built-in | Excellent |
| authorized_keys | Per-server | Manual sync | None | Poor |
| LDAP SSH keys | Centralized | Manual | External | Good |
| HashiCorp Vault | Centralized | Instant | Built-in | Excellent |

MageBox SSH CA provides enterprise-grade security without external dependencies.

## Troubleshooting

### Certificate Rejected

```
Permission denied (publickey)
```

Check:
1. Certificate not expired: `magebox cert show`
2. Renew if needed: `magebox cert renew`
3. CA key deployed on server: `cat /etc/ssh/magebox-ca.pub`
4. sshd configured: `grep TrustedUserCAKeys /etc/ssh/sshd_config`

### CA Not Trusted

```
Certificate not trusted
```

The server's CA public key doesn't match. Re-deploy:
```bash
magebox server env sync myapp/production --force-ca
```

### Clock Skew

Certificates are time-sensitive. Ensure clocks are synchronized:
```bash
# On client and server
timedatectl status
```

Use NTP for clock synchronization.

## Next Steps

- [Team Server Guide](/guide/team-server) - Full Team Server documentation
- [ISO 27001 Compliance](/guide/team-server-compliance) - Control mapping
