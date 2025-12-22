#!/bin/sh
set -e

DATA_DIR="/var/lib/magebox/teamserver"
DB_FILE="$DATA_DIR/teamserver.db"

# Check if already initialized (check for database file instead of config)
if [ ! -f "$DB_FILE" ]; then
    echo "Initializing server with SSH CA..."

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

    # Use magebox server init to properly initialize with CA keys
    /usr/local/bin/magebox server init \
        --data-dir "$DATA_DIR" \
        --admin-token "$ADMIN_TOKEN" \
        --master-key "$MASTER_KEY"

    echo "Server initialized with SSH CA enabled"
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
