/**
 * Created by Qoliber
 *
 * @category    Qoliber
 * @package     MageBox
 * @author      Jakub Winkler <jwinkler@qoliber.com>
 */

package teamserver

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Notifier handles email notifications
type Notifier struct {
	config    SMTPConfig
	from      string
	enabled   bool
	templates map[string]*template.Template
}

// NewNotifier creates a new email notifier
func NewNotifier(config SMTPConfig) *Notifier {
	n := &Notifier{
		config:    config,
		from:      config.From,
		enabled:   config.Enabled,
		templates: make(map[string]*template.Template),
	}

	if n.from == "" {
		n.from = "noreply@magebox.local"
	}

	// Initialize templates
	n.initTemplates()

	return n
}

// IsEnabled returns whether notifications are enabled
func (n *Notifier) IsEnabled() bool {
	return n.enabled && n.config.Host != ""
}

// initTemplates initializes email templates
func (n *Notifier) initTemplates() {
	// User invited template
	n.templates["user_invited"] = template.Must(template.New("user_invited").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>MageBox Team Invitation</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">MageBox Team Invitation</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <p>Hello <strong>{{.UserName}}</strong>,</p>
        <p>You have been invited to join the MageBox team with the role of <strong>{{.Role}}</strong>.</p>
        <p>To accept this invitation, run the following command:</p>
        <pre style="background: #2d2d2d; color: #f8f8f2; padding: 15px; border-radius: 5px; overflow-x: auto;">magebox team join {{.ServerURL}} --token {{.InviteToken}}</pre>
        <p style="color: #666; font-size: 14px;">This invitation expires on <strong>{{.ExpiresAt}}</strong>.</p>
        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">If you did not expect this invitation, please ignore this email.</p>
    </div>
</body>
</html>
`))

	// User joined template
	n.templates["user_joined"] = template.Must(template.New("user_joined").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome to MageBox Team</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">Welcome to MageBox Team!</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <p>Hello <strong>{{.UserName}}</strong>,</p>
        <p>Your account has been successfully created with the role of <strong>{{.Role}}</strong>.</p>
        {{if .Environments}}
        <p>You now have access to the following environments:</p>
        <ul>
            {{range .Environments}}
            <li>{{.}}</li>
            {{end}}
        </ul>
        {{end}}
        <p>You can check your access status anytime with:</p>
        <pre style="background: #2d2d2d; color: #f8f8f2; padding: 15px; border-radius: 5px; overflow-x: auto;">magebox team status</pre>
        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated message from MageBox Team Server.</p>
    </div>
</body>
</html>
`))

	// User removed template
	n.templates["user_removed"] = template.Must(template.New("user_removed").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>MageBox Access Revoked</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">MageBox Access Revoked</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <p>Hello <strong>{{.UserName}}</strong>,</p>
        <p>Your access to the MageBox team has been revoked.</p>
        <p>Your SSH keys have been removed from all environments.</p>
        <p style="color: #666;">If you believe this was done in error, please contact your administrator.</p>
        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated message from MageBox Team Server.</p>
    </div>
</body>
</html>
`))

	// Security alert template
	n.templates["security_alert"] = template.Must(template.New("security_alert").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>MageBox Security Alert</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #ff416c 0%, #ff4b2b 100%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">Security Alert</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <p><strong>Alert Type:</strong> {{.AlertType}}</p>
        <p><strong>Time:</strong> {{.Timestamp}}</p>
        <p><strong>IP Address:</strong> {{.IPAddress}}</p>
        <p><strong>Details:</strong> {{.Details}}</p>
        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated security alert from MageBox Team Server.</p>
    </div>
</body>
</html>
`))

	// Access expiry warning template
	n.templates["access_expiry"] = template.Must(template.New("access_expiry").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>MageBox Access Expiry Warning</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">Access Expiry Warning</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <p>Hello <strong>{{.UserName}}</strong>,</p>
        <p>Your MageBox team access will expire on <strong>{{.ExpiresAt}}</strong> ({{.DaysLeft}} days remaining).</p>
        <p>Please contact your administrator to renew your access if needed.</p>
        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated message from MageBox Team Server.</p>
    </div>
</body>
</html>
`))

	// Admin notification template
	n.templates["admin_notification"] = template.Must(template.New("admin_notification").Parse(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>MageBox Admin Notification</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); padding: 30px; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">Admin Notification</h1>
    </div>
    <div style="background: #f9f9f9; padding: 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <p><strong>Event:</strong> {{.Event}}</p>
        <p><strong>Time:</strong> {{.Timestamp}}</p>
        <p><strong>User:</strong> {{.UserName}}</p>
        <p><strong>Details:</strong> {{.Details}}</p>
        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 20px 0;">
        <p style="color: #999; font-size: 12px;">This is an automated admin notification from MageBox Team Server.</p>
    </div>
</body>
</html>
`))
}

// SendUserInvited sends an invitation email
func (n *Notifier) SendUserInvited(email, userName, role, serverURL, inviteToken string, expiresAt time.Time) error {
	if !n.IsEnabled() {
		return nil
	}

	data := map[string]interface{}{
		"UserName":    userName,
		"Role":        role,
		"ServerURL":   serverURL,
		"InviteToken": inviteToken,
		"ExpiresAt":   expiresAt.Format("January 2, 2006 at 15:04 MST"),
	}

	return n.sendTemplatedEmail(email, "MageBox Team Invitation", "user_invited", data)
}

// SendUserJoined sends a welcome email
func (n *Notifier) SendUserJoined(email, userName, role string, environments []string) error {
	if !n.IsEnabled() {
		return nil
	}

	data := map[string]interface{}{
		"UserName":     userName,
		"Role":         role,
		"Environments": environments,
	}

	return n.sendTemplatedEmail(email, "Welcome to MageBox Team", "user_joined", data)
}

// SendUserRemoved sends an access revoked email
func (n *Notifier) SendUserRemoved(email, userName string) error {
	if !n.IsEnabled() {
		return nil
	}

	data := map[string]interface{}{
		"UserName": userName,
	}

	return n.sendTemplatedEmail(email, "MageBox Access Revoked", "user_removed", data)
}

// SendSecurityAlert sends a security alert email
func (n *Notifier) SendSecurityAlert(email, alertType, ipAddress, details string) error {
	if !n.IsEnabled() {
		return nil
	}

	data := map[string]interface{}{
		"AlertType": alertType,
		"Timestamp": time.Now().Format("January 2, 2006 at 15:04:05 MST"),
		"IPAddress": ipAddress,
		"Details":   details,
	}

	return n.sendTemplatedEmail(email, "MageBox Security Alert: "+alertType, "security_alert", data)
}

// SendAccessExpiryWarning sends an access expiry warning email
func (n *Notifier) SendAccessExpiryWarning(email, userName string, expiresAt time.Time, daysLeft int) error {
	if !n.IsEnabled() {
		return nil
	}

	data := map[string]interface{}{
		"UserName":  userName,
		"ExpiresAt": expiresAt.Format("January 2, 2006"),
		"DaysLeft":  daysLeft,
	}

	return n.sendTemplatedEmail(email, "MageBox Access Expiry Warning", "access_expiry", data)
}

// SendAdminNotification sends a notification to admins
func (n *Notifier) SendAdminNotification(email, event, userName, details string) error {
	if !n.IsEnabled() {
		return nil
	}

	data := map[string]interface{}{
		"Event":     event,
		"Timestamp": time.Now().Format("January 2, 2006 at 15:04:05 MST"),
		"UserName":  userName,
		"Details":   details,
	}

	return n.sendTemplatedEmail(email, "MageBox Admin Notification: "+event, "admin_notification", data)
}

// sendTemplatedEmail renders a template and sends the email
func (n *Notifier) sendTemplatedEmail(to, subject, templateName string, data interface{}) error {
	tmpl, ok := n.templates[templateName]
	if !ok {
		return fmt.Errorf("template not found: %s", templateName)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return n.sendEmail(to, subject, body.String())
}

// sendEmail sends an email using SMTP
func (n *Notifier) sendEmail(to, subject, htmlBody string) error {
	// Build message
	headers := make(map[string]string)
	headers["From"] = n.from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	// Determine address (IPv6 compatible)
	addr := net.JoinHostPort(n.config.Host, strconv.Itoa(n.config.Port))

	// Choose authentication method
	var auth smtp.Auth
	if n.config.User != "" {
		auth = smtp.PlainAuth("", n.config.User, n.config.Password, n.config.Host)
	}

	// Send based on port (TLS vs STARTTLS vs plain)
	switch n.config.Port {
	case 465:
		// Implicit TLS (SMTPS)
		return n.sendWithTLS(addr, auth, to, msg.Bytes())
	case 587, 25:
		// STARTTLS or plain
		return n.sendWithSTARTTLS(addr, auth, to, msg.Bytes())
	default:
		// Try plain SMTP (for testing with Mailpit)
		return smtp.SendMail(addr, auth, n.from, []string{to}, msg.Bytes())
	}
}

// sendWithTLS sends email using implicit TLS (port 465)
func (n *Notifier) sendWithTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: n.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, n.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	return n.sendWithClient(client, auth, to, msg)
}

// sendWithSTARTTLS sends email using STARTTLS (port 587/25)
func (n *Notifier) sendWithSTARTTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Try STARTTLS if available
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: n.config.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	return n.sendWithClient(client, auth, to, msg)
}

// sendWithClient sends email using an existing SMTP client
func (n *Notifier) sendWithClient(client *smtp.Client, auth smtp.Auth, to string, msg []byte) error {
	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(n.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// TestConnection tests the SMTP connection
func (n *Notifier) TestConnection() error {
	if !n.IsEnabled() {
		return fmt.Errorf("email notifications are not enabled")
	}

	addr := net.JoinHostPort(n.config.Host, strconv.Itoa(n.config.Port))

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Check if server is responsive
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP HELO failed: %w", err)
	}

	return client.Quit()
}

// NotificationEvent represents a notification event type
type NotificationEvent string

const (
	NotifyUserInvited   NotificationEvent = "USER_INVITED"
	NotifyUserJoined    NotificationEvent = "USER_JOINED"
	NotifyUserRemoved   NotificationEvent = "USER_REMOVED"
	NotifySecurityAlert NotificationEvent = "SECURITY_ALERT"
	NotifyAccessExpiry  NotificationEvent = "ACCESS_EXPIRY"
	NotifyKeyDeployed   NotificationEvent = "KEY_DEPLOYED"
	NotifyKeyRemoved    NotificationEvent = "KEY_REMOVED"
)

// String returns the string representation of the event
func (e NotificationEvent) String() string {
	return string(e)
}

// GetAdminEmails returns admin email addresses
func GetAdminEmails(users []User) []string {
	var emails []string
	for _, u := range users {
		if u.Role == RoleAdmin && u.Email != "" {
			emails = append(emails, u.Email)
		}
	}
	return emails
}

// SanitizeEmail performs basic email validation
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	return email
}
