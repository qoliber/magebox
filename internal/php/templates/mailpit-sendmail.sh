#!/bin/bash
# MageBox Mailpit sendmail wrapper
# Routes PHP mail() calls to Mailpit SMTP server
#
# Usage: This script is set as PHP's sendmail_path when Mailpit is enabled
# It reads the email from stdin and sends it via SMTP to Mailpit

MAILPIT_HOST="${MAILPIT_HOST:-127.0.0.1}"
MAILPIT_PORT="${MAILPIT_PORT:-1025}"

# Read the entire email from stdin
EMAIL_CONTENT=$(cat)

# Extract headers and body
# The email content comes in standard RFC 822 format from PHP

# Use curl to send via SMTP (if available) or fall back to netcat
if command -v curl &> /dev/null; then
    # curl supports SMTP since version 7.20
    echo "$EMAIL_CONTENT" | curl --silent --show-error \
        --url "smtp://${MAILPIT_HOST}:${MAILPIT_PORT}" \
        --mail-from "$(echo "$EMAIL_CONTENT" | grep -i '^From:' | head -1 | sed 's/^From: *//i' | sed 's/.*<\(.*\)>.*/\1/')" \
        --mail-rcpt "$(echo "$EMAIL_CONTENT" | grep -i '^To:' | head -1 | sed 's/^To: *//i' | sed 's/.*<\(.*\)>.*/\1/')" \
        --upload-file - 2>/dev/null
    exit $?
fi

# Fallback: use PHP's built-in socket if curl is not available
# This creates a simple SMTP conversation
php -r '
$host = getenv("MAILPIT_HOST") ?: "127.0.0.1";
$port = getenv("MAILPIT_PORT") ?: 1025;
$email = file_get_contents("php://stdin");

// Extract From and To from headers
preg_match("/^From:\s*(.+)$/mi", $email, $from_match);
preg_match("/^To:\s*(.+)$/mi", $email, $to_match);

$from = isset($from_match[1]) ? trim($from_match[1]) : "noreply@localhost";
$to = isset($to_match[1]) ? trim($to_match[1]) : "test@localhost";

// Extract email from "Name <email>" format
if (preg_match("/<([^>]+)>/", $from, $m)) $from = $m[1];
if (preg_match("/<([^>]+)>/", $to, $m)) $to = $m[1];

$socket = @fsockopen($host, $port, $errno, $errstr, 5);
if (!$socket) {
    fwrite(STDERR, "Could not connect to Mailpit: $errstr ($errno)\n");
    exit(1);
}

// Simple SMTP conversation
fgets($socket); // Read greeting
fwrite($socket, "EHLO localhost\r\n"); fgets($socket);
fwrite($socket, "MAIL FROM:<$from>\r\n"); fgets($socket);
fwrite($socket, "RCPT TO:<$to>\r\n"); fgets($socket);
fwrite($socket, "DATA\r\n"); fgets($socket);
fwrite($socket, $email . "\r\n.\r\n"); fgets($socket);
fwrite($socket, "QUIT\r\n");
fclose($socket);
' <<< "$EMAIL_CONTENT"
