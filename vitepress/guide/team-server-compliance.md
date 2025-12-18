# ISO 27001 Compliance

The MageBox Team Server is designed to help organizations meet ISO 27001 requirements for information security management.

## Understanding ISO 27001

ISO 27001 is a **framework** for Information Security Management Systems (ISMS), not a technical checklist. It requires:

1. **Policies and procedures** - Documented security policies
2. **Controls appropriate to your risk assessment** - You choose which controls apply
3. **Evidence that controls are working** - Audit trails and reviews

Organizations select applicable controls via a **Statement of Applicability (SoA)**. Not every control is mandatory - you implement what's appropriate for your risk profile.

## What the Team Server Provides

The following table shows security capabilities implemented in the Team Server:

| Requirement | Implementation | Status |
|-------------|----------------|--------|
| Access control policy | Project-based grants with explicit approval | Implemented |
| User registration | Formal invite flow with admin approval | Implemented |
| Access provisioning | `user grant` / `user revoke` commands | Implemented |
| Privileged access management | Admin role with optional MFA requirement | Implemented |
| Access revocation | Instant removal via `user remove` | Implemented |
| Encryption at rest | AES-256-GCM for SSH keys, MFA secrets, tokens | Implemented |
| Encryption in transit | TLS support for all API communications | Implemented |
| Token security | Argon2id hashing (memory-hard algorithm) | Implemented |
| MFA support | TOTP (Google Authenticator compatible) | Implemented |
| Audit trail | All actions logged with timestamp, user, IP | Implemented |
| Tamper-evident logs | Hash chain verification | Implemented |
| Log export | JSON/CSV formats for auditors | Implemented |
| Log retention | Configurable (default 365 days) | Implemented |
| Security headers | X-Content-Type-Options, X-Frame-Options, etc. | Implemented |
| IP lockout | After failed login attempts | Implemented |
| Email notifications | Security alerts, invite notifications | Implemented |

## Nice-to-Have Features (Not Required for Compliance)

These features would provide automation convenience but are **not required** for ISO 27001 certification. Manual equivalents are acceptable:

| Feature | Manual Alternative | Status |
|---------|-------------------|--------|
| Automated compliance reports | Export audit logs manually (`magebox server audit --format json`) | Not implemented |
| Automatic expiry warnings | Review user list periodically (`magebox server user list`) | Not implemented |
| Access review command | Query audit logs and user access manually | Not implemented |
| Automatic key rotation | Rotate keys manually when needed | Not implemented |

::: warning Important
ISO 27001 requires that you HAVE controls, not that they be automated. Manual processes are fully acceptable as long as they are documented and followed consistently.
:::

## Control Mapping

Below is a mapping of implemented features to relevant ISO 27001 Annex A controls.

### A.9 - Access Control

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

### A.10 - Cryptography

| Control | Implementation |
|---------|----------------|
| A.10.1.1 Cryptographic controls | AES-256-GCM for data at rest, TLS for data in transit |
| A.10.1.2 Key management | Master key for encryption, secure token generation |

### A.12 - Operations Security

| Control | Implementation |
|---------|----------------|
| A.12.4.1 Event logging | All security events logged with timestamp, user, IP |
| A.12.4.2 Protection of log info | Tamper-evident hash chain for audit integrity |
| A.12.4.3 Admin logs | All admin actions logged separately |

### A.13 - Communications Security

| Control | Implementation |
|---------|----------------|
| A.13.1.1 Network controls | TLS encryption, configurable allowed IPs |
| A.13.2.1 Information transfer | SSH keys deployed over encrypted SSH connections |

### A.18 - Compliance

| Control | Implementation |
|---------|----------------|
| A.18.1.3 Protection of records | Audit log retention (configurable, default 365 days) |
| A.18.1.4 Privacy | Sensitive data encrypted, tokens never exposed in logs |

## Compliance Checklist

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

## Audit Report Export

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

## Recommended Procedures

To achieve ISO 27001 compliance, document and follow these procedures:

### Quarterly Access Review

```bash
# 1. List all users and their project access
magebox server user list

# 2. For each user, verify access is still required
magebox server user show <username>

# 3. Revoke access for users who no longer need it
magebox server user revoke <username> --project <project>

# 4. Document the review in your ISMS records
```

### Monthly Audit Log Review

```bash
# 1. Export last month's audit log
magebox server audit --from $(date -d "1 month ago" +%Y-%m-%d) --format json > monthly-audit.json

# 2. Verify audit chain integrity
magebox server audit verify

# 3. Review for anomalies (failed logins, unusual access patterns)
# 4. Document findings in your ISMS records
```

### Onboarding New Users

```bash
# 1. Create user invitation (requires admin approval)
magebox server user add <username> --email <email> --role <role>

# 2. Grant project access based on job requirements
magebox server user grant <username> --project <project>

# 3. User accepts invite and registers SSH key
magebox server join <server-url> --token <token> --key ~/.ssh/id_ed25519.pub

# 4. Document in your access management records
```

### Offboarding Users

```bash
# 1. Remove user (automatically revokes all access and removes SSH keys)
magebox server user remove <username>

# 2. Verify removal in audit log
magebox server audit --user <username> --format json

# 3. Document in your access management records
```

## Summary

::: tip Key Takeaways
1. **ISO 27001 is a framework**, not a technical checklist
2. **Manual processes are acceptable** - automation is convenient but not required
3. **Document your procedures** - the key is consistency
4. **The Team Server provides all necessary controls** for ISO 27001 compliance
5. **Export audit logs** regularly for compliance evidence
:::
