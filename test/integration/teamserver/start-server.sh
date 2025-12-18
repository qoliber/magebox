#!/bin/sh
set -e

DATA_DIR="/var/lib/magebox/teamserver"
CONFIG_FILE="$DATA_DIR/server.json"

# Check if already initialized
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Initializing server..."

    # Create config with provided credentials
    MASTER_KEY="${MAGEBOX_MASTER_KEY:-}"
    ADMIN_TOKEN="${MAGEBOX_ADMIN_TOKEN:-}"

    if [ -z "$MASTER_KEY" ]; then
        echo "ERROR: MAGEBOX_MASTER_KEY environment variable is required"
        exit 1
    fi

    if [ -z "$ADMIN_TOKEN" ]; then
        echo "ERROR: MAGEBOX_ADMIN_TOKEN environment variable is required"
        exit 1
    fi

    # Hash the admin token using a simple approach for testing
    # In production, the magebox server init command would be used
    cat > "$CONFIG_FILE" << EOF
{
    "master_key": "$MASTER_KEY",
    "admin_token_hash": "",
    "port": 7443,
    "host": "0.0.0.0",
    "tls_enabled": false,
    "initialized_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

    echo "Server initialized"
fi

echo "Starting MageBox Team Server..."

# Build SMTP args if configured
SMTP_ARGS=""
if [ -n "$MAGEBOX_SMTP_HOST" ]; then
    SMTP_ARGS="--smtp-host $MAGEBOX_SMTP_HOST"
    if [ -n "$MAGEBOX_SMTP_PORT" ]; then
        SMTP_ARGS="$SMTP_ARGS --smtp-port $MAGEBOX_SMTP_PORT"
    fi
    if [ -n "$MAGEBOX_SMTP_FROM" ]; then
        SMTP_ARGS="$SMTP_ARGS --smtp-from $MAGEBOX_SMTP_FROM"
    fi
    if [ -n "$MAGEBOX_SMTP_USER" ]; then
        SMTP_ARGS="$SMTP_ARGS --smtp-user $MAGEBOX_SMTP_USER"
    fi
    if [ -n "$MAGEBOX_SMTP_PASSWORD" ]; then
        SMTP_ARGS="$SMTP_ARGS --smtp-password $MAGEBOX_SMTP_PASSWORD"
    fi
fi

exec /usr/local/bin/magebox server start \
    --port 7443 \
    --host 0.0.0.0 \
    --data-dir "$DATA_DIR" \
    --admin-token "$MAGEBOX_ADMIN_TOKEN" \
    --master-key "$MAGEBOX_MASTER_KEY" \
    $SMTP_ARGS
