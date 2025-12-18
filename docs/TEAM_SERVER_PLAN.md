# MageBox Team Server - Implementation Plan

## Overview

A secure, ISO 27001-compliant team access management system built into MageBox. Allows organizations to manage SSH access to remote environments with role-based access control, audit logging, and automatic key distribution.

## Goals

- **Simple**: One binary, no external dependencies
- **Secure**: ISO 27001 / SOC 2 compliant
- **Self-hosted**: Runs on any VPS
- **No proxy**: Direct SSH connections, MageBox manages authorized_keys

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MageBox Team Server                       │
│                    (your VPS)                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   REST API   │  │   Storage    │  │    Deploy    │       │
│  │  (HTTPS)     │  │  (SQLite)    │  │   Service    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│         │                 │                 │                │
│         └─────────────────┴─────────────────┘                │
│                           │                                  │
│                    ┌──────┴──────┐                          │
│                    │   Audit     │                          │
│                    │   Logger    │                          │
│                    └─────────────┘                          │
└─────────────────────────────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
          ▼                 ▼                 ▼
    ┌──────────┐      ┌──────────┐      ┌──────────┐
    │  Admin   │      │   Dev    │      │ Readonly │
    │ (jakub)  │      │ (tomek)  │      │  (anna)  │
    └──────────┘      └──────────┘      └──────────┘
          │                 │
          │    Direct SSH   │
          ▼                 ▼
    ┌──────────┐      ┌──────────┐
    │Production│      │ Staging  │
    └──────────┘      └──────────┘
```

## Security Requirements (ISO 27001)

### Access Control (A.9)

- [ ] Role-based access control (admin, dev, readonly)
- [ ] Formal user registration with invite flow
- [ ] Access provisioning per environment
- [ ] Privileged access requires MFA (admin role)
- [ ] Periodic access review with expiry
- [ ] Instant access revocation

### Operations Security (A.12)

- [ ] All actions logged with timestamp, user, IP
- [ ] Tamper-evident logs (hash chain)
- [ ] Admin actions highlighted
- [ ] Log export for compliance audits

### Cryptography (A.10)

- [ ] TLS 1.3 for all communications
- [ ] AES-256-GCM for data at rest
- [ ] Argon2id for token hashing
- [ ] Automatic key rotation

---

## Implementation Phases

### Phase 1: Core Server & Storage

**Goal**: Basic server that can store users and environments

**Tasks**:
- [ ] Create `internal/teamserver/` package
- [ ] SQLite database schema (users, environments, access)
- [ ] Encryption layer for sensitive data
- [ ] Basic REST API structure
- [ ] Server start/stop commands

**Commands**:
```bash
magebox server start --port 7443 --admin-token "xxx"
magebox server stop
magebox server status
```

**Files**:
```
internal/teamserver/
├── server.go           # HTTP server
├── storage.go          # SQLite operations
├── crypto.go           # Encryption helpers
└── models.go           # Data structures

cmd/magebox/
└── server_cmd.go       # CLI commands
```

---

### Phase 2: User Management

**Goal**: Admin can create/remove users with roles

**Tasks**:
- [ ] Invite token generation (one-time, expiry)
- [ ] User registration flow
- [ ] Role definitions (admin, dev, readonly)
- [ ] Session token management
- [ ] Token hashing with Argon2id

**Commands**:
```bash
# Admin
magebox team user add <name> --email <email> --role <role>
magebox team user remove <name>
magebox team user list

# User
magebox team join <server-url> --token <invite-token>
magebox team status
```

**API Endpoints**:
```
POST /api/admin/users           # Create user (admin)
DELETE /api/admin/users/:name   # Remove user (admin)
GET /api/admin/users            # List users (admin)
POST /api/join                  # Accept invite (user)
GET /api/me                     # Current user info
```

---

### Phase 3: Environment Management

**Goal**: Define environments and access rules

**Tasks**:
- [ ] Environment configuration storage
- [ ] Deploy key management (encrypted)
- [ ] Role-to-environment mapping
- [ ] Environment status checking

**Commands**:
```bash
magebox team env add <name> --host <host> --deploy-user <user> --deploy-key <key> --roles <roles>
magebox team env remove <name>
magebox team env list
```

**API Endpoints**:
```
POST /api/admin/environments        # Add environment
DELETE /api/admin/environments/:name
GET /api/admin/environments         # List all (admin)
GET /api/environments               # List accessible (user)
```

---

### Phase 4: Key Distribution

**Goal**: Automatically deploy/revoke SSH keys to environments

**Tasks**:
- [ ] SSH connection to environments
- [ ] authorized_keys management
- [ ] Key deployment on user join
- [ ] Key removal on user removal
- [ ] Batch key sync

**Commands**:
```bash
magebox team sync                   # Sync keys to all environments
magebox team env sync <name>        # Sync specific environment
```

**Flow**:
```
User joins → Server deploys pubkey to allowed environments' authorized_keys
User removed → Server removes pubkey from all environments
```

---

### Phase 5: Audit Logging

**Goal**: Complete audit trail for compliance

**Tasks**:
- [ ] Event logging (all actions)
- [ ] Hash chain for tamper evidence
- [ ] Log retention policy
- [ ] Log export (CSV, JSON)
- [ ] Log search/filter

**Commands**:
```bash
magebox team audit [--from DATE] [--to DATE] [--user USER]
magebox team audit --export csv > audit.csv
```

**Logged Events**:
```
USER_CREATE, USER_REMOVE, USER_JOIN
ENV_CREATE, ENV_REMOVE, ENV_ACCESS
KEY_DEPLOYED, KEY_REMOVED, KEY_ROTATED
AUTH_SUCCESS, AUTH_FAILED
ADMIN_ACTION, CONFIG_CHANGE
```

---

### Phase 6: MFA & Security Hardening

**Goal**: Multi-factor authentication and advanced security

**Tasks**:
- [ ] TOTP support (Google Authenticator)
- [ ] MFA requirement for admin role
- [ ] Rate limiting
- [ ] IP allowlist (optional)
- [ ] Failed login alerts
- [ ] Security headers

**Commands**:
```bash
magebox team mfa setup              # Setup MFA
magebox team mfa verify <code>      # Verify MFA
```

---

### Phase 7: Notifications

**Goal**: Email and webhook notifications

**Tasks**:
- [ ] SMTP configuration
- [ ] Invite emails
- [ ] Access granted notifications
- [ ] Security alert emails
- [ ] Webhook support (Slack, Discord)

**Commands**:
```bash
magebox server config set smtp.host <host>
magebox server config set smtp.user <user>
magebox server config set smtp.password <password>
magebox server config set webhook.url <url>
```

**Notifications**:
- User invited
- User joined
- User removed
- Access changed
- Failed login attempts
- Key rotation

---

### Phase 8: Access Review & Compliance

**Goal**: Periodic review and compliance reporting

**Tasks**:
- [ ] Access expiry
- [ ] Access review command
- [ ] Compliance report generation
- [ ] Automatic expiry warnings

**Commands**:
```bash
magebox team review                 # Show access review
magebox team user renew <name> --expires 90d
magebox team compliance-report      # Generate report
```

---

## Database Schema

```sql
-- Users
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL,              -- admin, dev, readonly
    pubkey TEXT,                     -- SSH public key
    token_hash TEXT,                 -- Hashed session token
    mfa_secret TEXT,                 -- TOTP secret (encrypted)
    expires_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT
);

-- Environments
CREATE TABLE environments (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    host TEXT NOT NULL,
    port INTEGER DEFAULT 22,
    deploy_user TEXT NOT NULL,
    deploy_key TEXT NOT NULL,        -- Encrypted private key
    allowed_roles TEXT NOT NULL,     -- Comma-separated roles
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Invites
CREATE TABLE invites (
    id INTEGER PRIMARY KEY,
    token_hash TEXT UNIQUE NOT NULL,
    user_name TEXT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    environments TEXT,               -- Comma-separated env names
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Audit Log
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    user_name TEXT,
    action TEXT NOT NULL,
    details TEXT,
    ip_address TEXT,
    prev_hash TEXT,                  -- Hash chain
    hash TEXT NOT NULL
);

-- Config
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

---

## API Reference

### Authentication

All requests (except `/api/join`) require `Authorization: Bearer <token>` header.

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/admin/users | Create user & invite |
| GET | /api/admin/users | List all users |
| DELETE | /api/admin/users/:name | Remove user |
| PUT | /api/admin/users/:name | Update user |
| POST | /api/admin/environments | Add environment |
| GET | /api/admin/environments | List environments |
| DELETE | /api/admin/environments/:name | Remove environment |
| POST | /api/admin/sync | Sync all keys |
| GET | /api/admin/audit | Get audit log |

### User Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/join | Accept invite |
| GET | /api/me | Current user info |
| GET | /api/environments | List accessible environments |
| POST | /api/sync | Sync my access |
| POST | /api/mfa/setup | Setup MFA |
| POST | /api/mfa/verify | Verify MFA code |

---

## Configuration

```yaml
# /var/lib/magebox/server.yaml
server:
  port: 7443
  host: 0.0.0.0

security:
  tls:
    cert: /etc/magebox/cert.pem
    key: /etc/magebox/key.pem
    auto_tls: false
    domain: ""

  admin_token_hash: "<argon2id hash>"
  admin_mfa: required           # required, optional, disabled

  tokens:
    invite_expiry: 48h
    session_expiry: 30d

  rate_limit:
    enabled: true
    requests_per_minute: 60
    login_attempts: 5

  allowed_ips: []               # Empty = allow all

  encryption:
    algorithm: AES-256-GCM

notifications:
  smtp:
    enabled: false
    host: ""
    port: 587
    user: ""
    password: ""
    from: ""

  webhook:
    enabled: false
    url: ""

audit:
  retention_days: 365

access:
  default_expiry: 90d
  require_expiry: true
```

---

## Testing Plan

### Unit Tests
- [ ] Crypto functions (encrypt/decrypt, hash/verify)
- [ ] Token generation and validation
- [ ] Role permission checks
- [ ] Database operations

### Integration Tests
- [ ] Full invite → join flow
- [ ] Key deployment to test server
- [ ] Key removal on user delete
- [ ] Audit log integrity

### Security Tests
- [ ] TLS configuration
- [ ] Token brute force protection
- [ ] SQL injection prevention
- [ ] Unauthorized access attempts

---

## Timeline

| Phase | Description | Estimated Effort |
|-------|-------------|------------------|
| 1 | Core Server & Storage | Foundation |
| 2 | User Management | Core feature |
| 3 | Environment Management | Core feature |
| 4 | Key Distribution | Core feature |
| 5 | Audit Logging | Compliance |
| 6 | MFA & Security | Security |
| 7 | Notifications | Nice-to-have |
| 8 | Access Review | Compliance |

---

## Open Questions

1. **TLS Certificates**: Auto-generate with Let's Encrypt or require manual setup?
2. **Backup**: Should we include automated backup of the database?
3. **HA**: Any requirements for high availability / clustering?
4. **Integration**: Should we support LDAP/SAML for enterprise SSO?

---

## Next Steps

Start with **Phase 1: Core Server & Storage**:

1. Create `internal/teamserver/` package structure
2. Implement SQLite storage with encryption
3. Basic HTTP server with TLS
4. `magebox server start/stop/status` commands

Ready to begin?
