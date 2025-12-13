# DNS Configuration

MageBox supports two DNS modes for resolving `.test` domains.

## DNS Modes

### hosts Mode (Default)

Modifies `/etc/hosts` to resolve domains:

```
127.0.0.1 mystore.test
127.0.0.1 api.mystore.test
```

**Pros:**
- Simple and reliable
- No additional software needed
- Works everywhere

**Cons:**
- Requires sudo for each domain change
- Each domain must be added manually

### dnsmasq Mode

Uses dnsmasq for wildcard domain resolution:

```
*.test → 127.0.0.1
```

**Pros:**
- All `.test` domains resolve automatically
- No hosts file modifications
- Instant new domain support

**Cons:**
- Requires dnsmasq installation
- More complex setup

## Setting DNS Mode

### During Bootstrap

You'll be prompted to choose:

```bash
magebox bootstrap
# ? DNS mode: [hosts/dnsmasq]
```

### After Bootstrap

```bash
magebox config set dns_mode dnsmasq
magebox dns setup
```

## hosts Mode Setup

### Automatic

MageBox automatically updates `/etc/hosts` when you:

```bash
magebox start
```

### Manual

If automatic updates fail:

```bash
sudo nano /etc/hosts
```

Add:

```
127.0.0.1 mystore.test
127.0.0.1 another.test
```

## dnsmasq Mode Setup

### Installation

#### macOS

```bash
brew install dnsmasq
```

#### Ubuntu/Debian

```bash
sudo apt install dnsmasq
```

### Configuration

Run the setup command:

```bash
magebox dns setup
```

This configures dnsmasq to resolve `.test` domains.

### Manual Configuration

#### macOS

Create `/usr/local/etc/dnsmasq.d/test.conf`:

```
address=/test/127.0.0.1
```

Create resolver:

```bash
sudo mkdir -p /etc/resolver
echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/test
```

Start dnsmasq:

```bash
sudo brew services start dnsmasq
```

#### Linux

Linux DNS configuration varies by distribution and init system. Most modern distros use `systemd-resolved` which conflicts with dnsmasq on port 53.

##### Option 1: dnsmasq with systemd-resolved (Recommended)

This approach keeps systemd-resolved running but forwards `.test` queries to dnsmasq on an alternate port.

1. Install dnsmasq:

```bash
# Ubuntu/Debian
sudo apt install dnsmasq

# Fedora/RHEL
sudo dnf install dnsmasq
```

2. Configure dnsmasq to listen on a different port:

```bash
sudo tee /etc/dnsmasq.d/test.conf << 'EOF'
# Listen on localhost only, port 5354 (avoid conflict with systemd-resolved)
listen-address=127.0.0.1
port=5354

# Resolve all .test domains to localhost
address=/test/127.0.0.1

# Don't read /etc/resolv.conf
no-resolv

# Don't poll for changes
no-poll
EOF
```

3. Enable and start dnsmasq:

```bash
sudo systemctl enable dnsmasq
sudo systemctl start dnsmasq
```

4. Configure systemd-resolved to forward `.test` queries to dnsmasq:

```bash
sudo mkdir -p /etc/systemd/resolved.conf.d
sudo tee /etc/systemd/resolved.conf.d/test.conf << 'EOF'
[Resolve]
DNS=127.0.0.1:5354
Domains=~test
EOF
```

5. Restart systemd-resolved:

```bash
sudo systemctl restart systemd-resolved
```

##### Option 2: Replace systemd-resolved with dnsmasq

This approach disables systemd-resolved entirely and uses dnsmasq as the system DNS resolver.

1. Disable systemd-resolved:

```bash
sudo systemctl disable --now systemd-resolved
```

2. Remove the symlink and create a static resolv.conf:

```bash
sudo rm /etc/resolv.conf
sudo tee /etc/resolv.conf << 'EOF'
nameserver 127.0.0.1
EOF
```

3. Install and configure dnsmasq:

```bash
# Ubuntu/Debian
sudo apt install dnsmasq

# Fedora/RHEL
sudo dnf install dnsmasq
```

4. Create the test domain config:

```bash
sudo tee /etc/dnsmasq.d/test.conf << 'EOF'
address=/test/127.0.0.1
EOF
```

5. Configure upstream DNS servers:

```bash
sudo tee /etc/dnsmasq.d/upstream.conf << 'EOF'
# Use Cloudflare and Google as upstream DNS
server=1.1.1.1
server=8.8.8.8
EOF
```

6. Enable and start dnsmasq:

```bash
sudo systemctl enable dnsmasq
sudo systemctl start dnsmasq
```

##### Option 3: NetworkManager with dnsmasq plugin

If you use NetworkManager, you can use its built-in dnsmasq integration:

1. Configure NetworkManager to use dnsmasq:

```bash
sudo tee /etc/NetworkManager/conf.d/dnsmasq.conf << 'EOF'
[main]
dns=dnsmasq
EOF
```

2. Create the test domain config:

```bash
sudo mkdir -p /etc/NetworkManager/dnsmasq.d
sudo tee /etc/NetworkManager/dnsmasq.d/test.conf << 'EOF'
address=/test/127.0.0.1
EOF
```

3. Restart NetworkManager:

```bash
sudo systemctl restart NetworkManager
```

##### Verify Linux DNS Setup

```bash
# Check dnsmasq is running
systemctl status dnsmasq

# Test resolution
dig mystore.test @127.0.0.1

# If using alternate port (Option 1)
dig mystore.test @127.0.0.1 -p 5354

# Test through system resolver
ping mystore.test
```

## Checking DNS Status

```bash
magebox dns status
```

Output shows:
- Current DNS mode
- Configured domains
- Resolution status

## Testing Resolution

```bash
# Test domain resolution
ping mystore.test

# Test with dig (if installed)
dig mystore.test

# Test with nslookup
nslookup mystore.test
```

## Troubleshooting

### Domain Not Resolving

#### hosts Mode

1. Check hosts file:

```bash
cat /etc/hosts | grep test
```

2. Verify entry exists:

```
127.0.0.1 mystore.test
```

3. Try flushing DNS cache:

```bash
# macOS
sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder

# Linux
sudo systemctl restart systemd-resolved
```

#### dnsmasq Mode

1. Check dnsmasq is running:

```bash
# macOS
brew services list | grep dnsmasq

# Linux
systemctl status dnsmasq
```

2. Check resolver (macOS):

```bash
scutil --dns | grep test -A 5
```

3. Check dnsmasq config:

```bash
# macOS
cat /usr/local/etc/dnsmasq.d/test.conf

# Linux
cat /etc/dnsmasq.d/test.conf
```

### Conflicting DNS

If another service is using port 53:

```bash
sudo lsof -i :53
```

Common conflicts:
- systemd-resolved (Linux)
- Another dnsmasq instance
- VPN software

### Browser Not Using Local DNS

Some browsers have their own DNS settings:

#### Chrome

Disable "Use secure DNS":
1. Settings → Privacy and security
2. Security → Use secure DNS
3. Turn off or set to custom

#### Firefox

1. Settings → Network Settings
2. Uncheck "Enable DNS over HTTPS"

## Switching Modes

To switch from hosts to dnsmasq:

```bash
magebox config set dns_mode dnsmasq
magebox dns setup
```

To switch from dnsmasq to hosts:

```bash
magebox config set dns_mode hosts
# Domains will be added to /etc/hosts on next start
```

## Custom TLD

By default, MageBox uses `.test`. To change:

```bash
magebox config set tld local
```

Then reconfigure DNS:

```bash
magebox dns setup
```

::: warning
Changing TLD affects all projects and requires regenerating SSL certificates.
:::

## Recommended Approach

- **Single developer**: hosts mode is simpler
- **Many projects**: dnsmasq saves time with automatic resolution
- **Team environment**: Document your choice for consistency
