# MageBox Team Server

A centralized team access management system for secure SSH key distribution across environments. Designed with ISO 27001 compliance in mind.

## Overview

The MageBox Team Server provides:

- **Centralized User Management** - Invite users, assign roles, manage access
- **SSH Key Distribution** - Automatically deploy SSH keys to environments
- **Project-Based Access Control** - Users granted access to projects, not individual environments
- **Multi-Factor Authentication** - TOTP/MFA support for enhanced security
- **Audit Logging** - Tamper-evident audit trail with hash chain verification
- **Email Notifications** - Automated emails for invitations, security alerts
- **Security Features** - IP lockout, login attempt tracking

## Quick Start

### 1. Initialize the Server

```bash
magebox server init --data-dir /var/lib/magebox/teamserver
```

This generates:
- A **master encryption key** (save this securely!)
- An **admin token** for API authentication
- The configuration file

### 2. Start the Server

```bash
magebox server start \
    --port 7443 \
    --data-dir /var/lib/magebox/teamserver \
    --admin-token YOUR_ADMIN_TOKEN \
    --master-key YOUR_MASTER_KEY
```

Or with environment variables:

```bash
export MAGEBOX_ADMIN_TOKEN="your-admin-token"
export MAGEBOX_MASTER_KEY="your-64-char-hex-master-key"
magebox server start --port 7443
```

### 3. Create Projects

Projects are containers for environments. Users are granted access to projects.

```bash
# Create a project
magebox server project add myproject --description "My Application"
```

### 4. Add Environments

```bash
# Add environment to a project
magebox server env add staging \
    --project myproject \
    --host staging.example.com \
    --port 22 \
    --deploy-user deploy \
    --deploy-key ~/.ssh/staging_deploy_key
```

### 5. Invite Users

```bash
magebox server user add alice \
    --email alice@example.com \
    --role dev
```

This creates an invite token. Alice receives an email with instructions.

### 6. Grant Project Access

```bash
# Grant user access to a project
magebox server user grant alice --project myproject
```

### 7. User Joins

```bash
magebox server join https://teamserver.example.com \
    --token INVITE_TOKEN \
    --key ~/.ssh/id_ed25519.pub
```

Alice's SSH key is now automatically deployed to all environments she has access to.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Team Server                                 │
│                                                                  │
│  ┌────────────┐    ┌─────────────┐    ┌──────────────────────┐  │
│  │            │    │             │    │                      │  │
│  │   REST     │───▶│   Storage   │    │     Environments     │  │
│  │   API      │    │  (SQLite)   │    │                      │  │
│  │            │    │             │    │  ┌────────────────┐  │  │
│  └────────────┘    └─────────────┘    │  │   Staging      │  │  │
│        │                              │  │   (SSH)        │  │  │
│        │           ┌─────────────┐    │  └────────────────┘  │  │
│        │           │             │    │  ┌────────────────┐  │  │
│        └──────────▶│  Deployer   │───▶│  │   Production   │  │  │
│                    │ (SSH Keys)  │    │  │   (SSH)        │  │  │
│                    │             │    │  └────────────────┘  │  │
│                    └─────────────┘    └──────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## API Reference

### Authentication

All admin endpoints require Bearer token authentication:

```bash
curl -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
     https://teamserver.example.com/api/admin/users
```

User endpoints use session tokens obtained after joining.

### Admin Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/admin/users` | GET | List all users |
| `/api/admin/users` | POST | Create user invitation |
| `/api/admin/users/{name}` | GET | Get user details |
| `/api/admin/users/{name}` | DELETE | Remove user |
| `/api/admin/users/{name}/access` | POST | Grant project access |
| `/api/admin/users/{name}/access` | DELETE | Revoke project access |
| `/api/admin/projects` | GET | List all projects |
| `/api/admin/projects` | POST | Create project |
| `/api/admin/projects/{name}` | GET | Get project details |
| `/api/admin/projects/{name}` | DELETE | Delete project |
| `/api/admin/environments` | GET | List all environments |
| `/api/admin/environments` | POST | Add environment |
| `/api/admin/environments/{project}/{name}` | GET | Get environment |
| `/api/admin/environments/{project}/{name}` | DELETE | Remove environment |
| `/api/admin/audit` | GET | View audit log |
| `/api/admin/sync` | POST | Sync SSH keys |

### User Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/join` | POST | Accept invitation |
| `/api/me` | GET | Get current user info |
| `/api/environments` | GET | List accessible environments |
| `/api/mfa/setup` | GET | Get MFA setup (secret + QR) |
| `/api/mfa/setup` | POST | Confirm MFA with code |

### Public Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |

## Access Control Model

### Projects and Environments

Access is controlled at the **project** level, not the environment level:

```
Project: myproject
├── Environment: staging
├── Environment: qa
└── Environment: production
```

When a user is granted access to a project, they can access ALL environments within that project.

### User Roles

| Role | Description | Permissions |
|------|-------------|-------------|
| `admin` | Full access | Manage users, projects, environments |
| `dev` | Developer | Access granted projects |
| `readonly` | Read-only | View-only access |

### Managing Access

```bash
# Grant user access to a project
magebox server user grant alice --project myproject

# Revoke user access from a project
magebox server user revoke alice --project myproject

# List users with access to a project
magebox server project show myproject
```

## Multi-Factor Authentication

### Setup MFA

Users can enable MFA after joining:

```bash
# 1. Get setup info (secret + QR code)
curl -H "Authorization: Bearer SESSION_TOKEN" \
     https://teamserver.example.com/api/mfa/setup

# Response:
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/MageBox:alice@example.com?secret=..."
}

# 2. Scan QR code with authenticator app

# 3. Confirm with code
curl -X POST \
     -H "Authorization: Bearer SESSION_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"code": "123456"}' \
     https://teamserver.example.com/api/mfa/setup
```

### Admin MFA Requirement

For high-security environments, require MFA for admin operations:

```bash
magebox server start --require-admin-mfa
```

## Email Notifications

Configure SMTP for email notifications:

```bash
magebox server start \
    --smtp-host smtp.example.com \
    --smtp-port 587 \
    --smtp-user noreply@example.com \
    --smtp-password secret \
    --smtp-from noreply@example.com
```

Or via environment variables:

```bash
export MAGEBOX_SMTP_HOST=smtp.example.com
export MAGEBOX_SMTP_PORT=587
export MAGEBOX_SMTP_USER=noreply@example.com
export MAGEBOX_SMTP_PASSWORD=secret
export MAGEBOX_SMTP_FROM=noreply@example.com
```

### Email Types

| Event | Recipients | Description |
|-------|------------|-------------|
| User Invited | New user | Invitation with join instructions |
| User Joined | New user | Welcome email with access info |
| User Removed | Removed user | Access revocation notice |
| Security Alert | Admins | Failed login attempts, IP lockouts |
| Access Expiry | User | Warning before access expires |

## Audit Logging

All security-relevant actions are logged with a tamper-evident hash chain.

### View Audit Log

```bash
# All entries
magebox server audit

# Filter by user
magebox server audit --user alice

# Filter by action
magebox server audit --action USER_CREATE

# Date range
magebox server audit --from 2024-01-01 --to 2024-12-31

# Export formats
magebox server audit --format json
magebox server audit --format csv > audit.csv
```

### Verify Integrity

```bash
magebox server audit verify
```

This checks the hash chain to detect any tampering.

### Audit Actions

| Action | Description |
|--------|-------------|
| `USER_CREATE` | User invitation created |
| `USER_JOIN` | User accepted invitation |
| `USER_REMOVE` | User removed |
| `ENV_CREATE` | Environment added |
| `ENV_REMOVE` | Environment removed |
| `KEY_DEPLOY` | SSH key deployed |
| `KEY_REMOVE` | SSH key removed |
| `AUTH_SUCCESS` | Successful authentication |
| `AUTH_FAILED` | Failed authentication |
| `MFA_ENABLE` | MFA enabled |
| `IP_LOCKOUT` | IP locked due to failed attempts |

## Security Features

### IP Lockout

After 5 failed login attempts, the IP is locked for 15 minutes:

```json
{
  "error": "Too many failed login attempts. Please try again later.",
  "code": "IP_LOCKED"
}
```

### Security Headers

All responses include security headers:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`

### Encryption

- All sensitive data (SSH keys, MFA secrets) is encrypted at rest
- AES-256-GCM encryption with the master key
- Tokens are hashed with Argon2id

## ISO 27001 Compliance

The MageBox Team Server is designed to help organizations meet ISO 27001 requirements for information security management.

### Understanding ISO 27001

ISO 27001 is a **framework** for Information Security Management Systems (ISMS), not a technical checklist. It requires:

1. **Policies and procedures** - Documented security policies
2. **Controls appropriate to your risk assessment** - You choose which controls apply
3. **Evidence that controls are working** - Audit trails and reviews

Organizations select applicable controls via a **Statement of Applicability (SoA)**. Not every control is mandatory - you implement what's appropriate for your risk profile.

### What the Team Server Provides

The following table shows security capabilities implemented in the Team Server:

| Requirement | Implementation | Status |
|-------------|----------------|--------|
| Access control policy | Project-based grants with explicit approval | ✅ Implemented |
| User registration | Formal invite flow with admin approval | ✅ Implemented |
| Access provisioning | `user grant` / `user revoke` commands | ✅ Implemented |
| Privileged access management | Admin role with optional MFA requirement | ✅ Implemented |
| Access revocation | Instant removal via `user remove` | ✅ Implemented |
| Encryption at rest | AES-256-GCM for SSH keys, MFA secrets, tokens | ✅ Implemented |
| Encryption in transit | TLS support for all API communications | ✅ Implemented |
| Token security | Argon2id hashing (memory-hard algorithm) | ✅ Implemented |
| MFA support | TOTP (Google Authenticator compatible) | ✅ Implemented |
| Audit trail | All actions logged with timestamp, user, IP | ✅ Implemented |
| Tamper-evident logs | Hash chain verification | ✅ Implemented |
| Log export | JSON/CSV formats for auditors | ✅ Implemented |
| Log retention | Configurable (default 365 days) | ✅ Implemented |
| Security headers | X-Content-Type-Options, X-Frame-Options, etc. | ✅ Implemented |
| IP lockout | After failed login attempts | ✅ Implemented |
| Email notifications | Security alerts, invite notifications | ✅ Implemented |

### Nice-to-Have Features (Not Required for Compliance)

These features would provide automation convenience but are **not required** for ISO 27001 certification. Manual equivalents are acceptable:

| Feature | Manual Alternative | Status |
|---------|-------------------|--------|
| Automated compliance reports | Export audit logs manually (`magebox server audit --format json`) | Not implemented |
| Automatic expiry warnings | Review user list periodically (`magebox server user list`) | Not implemented |
| Access review command | Query audit logs and user access manually | Not implemented |
| Automatic key rotation | Rotate keys manually when needed | Not implemented |

**Important**: ISO 27001 requires that you HAVE controls, not that they be automated. Manual processes are fully acceptable as long as they are documented and followed consistently.

### Control Mapping

Below is a mapping of implemented features to relevant ISO 27001 Annex A controls.

#### A.9 - Access Control

| Control | Implementation |
|---------|----------------|
| A.9.1.1 Access control policy | Project-based access model with explicit grants |
| A.9.2.1 User registration | Formal invite flow with admin approval |
| A.9.2.2 User access provisioning | Project access granted via `user grant` command |
| A.9.2.3 Privileged access | Admin role with MFA requirement option (`--require-admin-mfa`) |
| A.9.2.5 Review of user access | Access expiry with configurable duration |
| A.9.2.6 Removal of access | Instant revocation via `user revoke` or `user remove` |
| A.9.4.1 Information access | Users only access environments in granted projects |
| A.9.4.2 Secure log-on | Token-based authentication with rate limiting |
| A.9.4.3 Password management | Tokens hashed with Argon2id (memory-hard algorithm) |

#### A.10 - Cryptography

| Control | Implementation |
|---------|----------------|
| A.10.1.1 Cryptographic controls | AES-256-GCM for data at rest, TLS for data in transit |
| A.10.1.2 Key management | Master key for encryption, secure token generation |

#### A.12 - Operations Security

| Control | Implementation |
|---------|----------------|
| A.12.4.1 Event logging | All security events logged with timestamp, user, IP |
| A.12.4.2 Protection of log info | Tamper-evident hash chain for audit integrity |
| A.12.4.3 Admin logs | All admin actions logged separately |

#### A.13 - Communications Security

| Control | Implementation |
|---------|----------------|
| A.13.1.1 Network controls | TLS encryption, configurable allowed IPs |
| A.13.2.1 Information transfer | SSH keys deployed over encrypted SSH connections |

#### A.18 - Compliance

| Control | Implementation |
|---------|----------------|
| A.18.1.3 Protection of records | Audit log retention (configurable, default 365 days) |
| A.18.1.4 Privacy | Sensitive data encrypted, tokens never exposed in logs |

### Compliance Checklist

Use this checklist to verify your deployment meets ISO 27001 requirements:

- [ ] **TLS enabled** - Enable TLS with valid certificates (`--tls-cert`, `--tls-key`)
- [ ] **Admin MFA required** - Enable `--require-admin-mfa` for privileged access
- [ ] **Master key secured** - Store master key in secrets manager (Vault, AWS Secrets Manager)
- [ ] **Audit log retention** - Configure retention per your policy (default 365 days)
- [ ] **Access expiry** - Set `default_access_days` to enforce periodic review
- [ ] **Email alerts** - Configure SMTP for security notifications
- [ ] **Network isolation** - Deploy in private network with firewall rules
- [ ] **Regular backups** - Backup SQLite database to secure location
- [ ] **Audit log review** - Export and review logs periodically (`magebox server audit`)
- [ ] **Integrity verification** - Verify audit chain regularly (`magebox server audit verify`)

### Audit Report Export

For compliance audits, export the audit log in your preferred format:

```bash
# Export as JSON for SIEM integration
magebox server audit --format json > audit-$(date +%Y%m%d).json

# Export as CSV for spreadsheet analysis
magebox server audit --format csv > audit-$(date +%Y%m%d).csv

# Filter by date range
magebox server audit --from 2025-01-01 --to 2025-12-31 --format json

# Filter by specific user
magebox server audit --user alice --format json
```

### Recommended Procedures

To achieve ISO 27001 compliance, document and follow these procedures:

#### Quarterly Access Review

```bash
# 1. List all users and their project access
magebox server user list

# 2. For each user, verify access is still required
magebox server user show <username>

# 3. Revoke access for users who no longer need it
magebox server user revoke <username> --project <project>

# 4. Document the review in your ISMS records
```

#### Monthly Audit Log Review

```bash
# 1. Export last month's audit log
magebox server audit --from $(date -d "1 month ago" +%Y-%m-%d) --format json > monthly-audit.json

# 2. Verify audit chain integrity
magebox server audit verify

# 3. Review for anomalies (failed logins, unusual access patterns)
# 4. Document findings in your ISMS records
```

#### Onboarding New Users

```bash
# 1. Create user invitation (requires admin approval)
magebox server user add <username> --email <email> --role <role>

# 2. Grant project access based on job requirements
magebox server user grant <username> --project <project>

# 3. User accepts invite and registers SSH key
magebox server join <server-url> --token <token> --key ~/.ssh/id_ed25519.pub

# 4. Document in your access management records
```

#### Offboarding Users

```bash
# 1. Remove user (automatically revokes all access and removes SSH keys)
magebox server user remove <username>

# 2. Verify removal in audit log
magebox server audit --user <username> --format json

# 3. Document in your access management records
```

## CLI Reference

### Server Commands

```bash
# Initialize server
magebox server init [--data-dir PATH]

# Start server
magebox server start [OPTIONS]
  --port PORT            Listen port (default: 7443)
  --host HOST            Listen address (default: 0.0.0.0)
  --data-dir PATH        Data directory
  --admin-token TOKEN    Admin authentication token
  --master-key KEY       Master encryption key (64 hex chars)
  --tls-cert FILE        TLS certificate file
  --tls-key FILE         TLS key file
  --smtp-host HOST       SMTP server host
  --smtp-port PORT       SMTP server port
  --smtp-user USER       SMTP username
  --smtp-password PASS   SMTP password
  --smtp-from EMAIL      From address for emails

# Stop server
magebox server stop

# Server status
magebox server status
```

### User Management

```bash
# Create invitation
magebox server user add USERNAME \
    --email EMAIL \
    --role ROLE

# List users
magebox server user list

# Show user details
magebox server user show USERNAME

# Remove user
magebox server user remove USERNAME

# Grant project access
magebox server user grant USERNAME --project PROJECT

# Revoke project access
magebox server user revoke USERNAME --project PROJECT
```

### Project Management

```bash
# Create project
magebox server project add NAME [--description DESC]

# List projects
magebox server project list

# Show project details (includes environments and users)
magebox server project show NAME

# Remove project (also removes all environments)
magebox server project remove NAME
```

### Environment Management

```bash
# Add environment to a project
magebox server env add NAME \
    --project PROJECT \
    --host HOSTNAME \
    --port PORT \
    --deploy-user USERNAME \
    --deploy-key PATH

# List environments
magebox server env list

# Show environment details
magebox server env show PROJECT/NAME

# Remove environment
magebox server env remove PROJECT/NAME

# Sync SSH keys to environments
magebox server env sync [PROJECT/NAME]
```

### Client Commands

```bash
# Join team server
magebox server join URL \
    --token INVITE_TOKEN \
    [--key PATH_TO_PUBLIC_KEY]

# Check status
magebox server whoami
```

## Docker Deployment

### docker-compose.yml

```yaml
version: '3.8'

services:
  teamserver:
    image: ghcr.io/qoliber/magebox:latest
    command: server start --port 7443 --data-dir /data
    ports:
      - "7443:7443"
    volumes:
      - teamserver-data:/data
    environment:
      - MAGEBOX_ADMIN_TOKEN=${ADMIN_TOKEN}
      - MAGEBOX_MASTER_KEY=${MASTER_KEY}
      - MAGEBOX_SMTP_HOST=mailserver
      - MAGEBOX_SMTP_PORT=25
      - MAGEBOX_SMTP_FROM=noreply@example.com
    restart: unless-stopped

volumes:
  teamserver-data:
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `MAGEBOX_ADMIN_TOKEN` | Admin API token |
| `MAGEBOX_MASTER_KEY` | 64-char hex encryption key |
| `MAGEBOX_SMTP_HOST` | SMTP server hostname |
| `MAGEBOX_SMTP_PORT` | SMTP server port |
| `MAGEBOX_SMTP_USER` | SMTP username |
| `MAGEBOX_SMTP_PASSWORD` | SMTP password |
| `MAGEBOX_SMTP_FROM` | Email from address |

## Integration Testing

Run the integration tests with Docker:

```bash
cd test/integration/teamserver
./run-tests.sh
```

This starts:
- Team server container
- Mailpit for email testing
- SSH test containers (staging, production)

Tests cover:
- User lifecycle (invite → join → remove)
- Environment management
- Role-based access control
- MFA setup and verification
- Email notifications
- Audit logging
- Security headers
- Concurrent access

## Troubleshooting

### Server won't start

```bash
# Check if port is in use
lsof -i :7443

# Check logs
journalctl -u magebox-teamserver
```

### Can't connect to environments

```bash
# Test SSH connection manually
ssh -i deploy_key deploy@staging.example.com

# Check firewall
sudo ufw status
```

### Emails not sending

```bash
# Test SMTP connection
telnet smtp.example.com 587

# Check SMTP credentials
curl -v smtps://smtp.example.com:465 --user user:pass
```

### Audit verification fails

If audit verification fails, the audit log may have been tampered with. Investigate:

```bash
# Export audit log for analysis
magebox server audit --format json > audit.json

# Check specific entries
jq '.[] | select(.id == 123)' audit.json
```

## Best Practices

1. **Secure the master key** - Store in a secrets manager (Vault, AWS Secrets Manager)
2. **Use TLS** - Always enable TLS in production
3. **Require MFA for admins** - Use `--require-admin-mfa`
4. **Regular audit reviews** - Export and review audit logs periodically
5. **Rotate admin tokens** - Change admin tokens regularly
6. **Network isolation** - Run team server in a private network
7. **Backup database** - Regular backups of the SQLite database
