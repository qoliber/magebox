#!/bin/sh
set -e

# Write environment info
echo "ENV_NAME=${ENV_NAME:-unknown}" > /etc/magebox/env-info
echo "DEPLOY_USER=${DEPLOY_USER:-deploy}" >> /etc/magebox/env-info
echo "HOSTNAME=$(hostname)" >> /etc/magebox/env-info

echo "Starting SSH server for environment: ${ENV_NAME:-unknown}"

# Ensure authorized_keys file exists with correct permissions
touch /home/deploy/.ssh/authorized_keys
chmod 600 /home/deploy/.ssh/authorized_keys
chown deploy:deploy /home/deploy/.ssh/authorized_keys

# Start SSH daemon in foreground
exec /usr/sbin/sshd -D -e
