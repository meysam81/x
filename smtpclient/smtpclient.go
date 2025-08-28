package smtpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client represents an SMTP client
type Client struct {
	host       string
	port       int
	username   string
	password   string
	authType   authType
	tlsConfig  *tls.Config
	timeout    time.Duration
	retryCount int
	retryDelay time.Duration
	skipVerify bool
	forceTLS   bool
	debug      bool
}

// Email represents an email message
type Email struct {
	From        string
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	HTMLBody    string
	TextBody    string
	Attachments []Attachment
	Headers     map[string]string
}

// Attachment represents an email attachment
type Attachment struct {
	Filename  string
	Content   io.Reader
	Inline    bool
	ContentID string
}

type authType int

const (
	authPlain authType = iota
	authLogin
	authCRAMMD5
	authNone
)

// Option configures the client
type Option func(*Client)

// New creates a new SMTP client with required parameters
func New(host string, port int, username, password string, opts ...Option) (*Client, error) {
	if host == "" {
		return nil, errors.New("host is required")
	}
	if port <= 0 || port > 65535 {
		return nil, errors.New("invalid port number")
	}

	c := &Client{
		host:       host,
		port:       port,
		username:   username,
		password:   password,
		authType:   authPlain,
		timeout:    30 * time.Second,
		retryCount: 3,
		retryDelay: time.Second,
		skipVerify: false,
		forceTLS:   true,
		debug:      false,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.tlsConfig == nil {
		c.tlsConfig = &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: c.skipVerify,
		}
	}

	return c, nil
}

// WithTimeout sets the connection timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(config *tls.Config) Option {
	return func(c *Client) {
		c.tlsConfig = config
	}
}

// WithInsecureSkipVerify skips TLS certificate verification (not recommended for production)
func WithInsecureSkipVerify() Option {
	return func(c *Client) {
		c.skipVerify = true
		if c.tlsConfig != nil {
			c.tlsConfig.InsecureSkipVerify = true
		}
	}
}

// WithAuthType sets the authentication type
func WithAuthType(authType string) Option {
	return func(c *Client) {
		switch strings.ToUpper(authType) {
		case "LOGIN":
			c.authType = authLogin
		case "CRAM-MD5":
			c.authType = authCRAMMD5
		case "NONE":
			c.authType = authNone
		default:
			c.authType = authPlain
		}
	}
}

// WithRetry configures retry behavior
func WithRetry(count int, delay time.Duration) Option {
	return func(c *Client) {
		c.retryCount = count
		c.retryDelay = delay
	}
}

// WithoutTLS disables TLS/STARTTLS
func WithoutTLS() Option {
	return func(c *Client) {
		c.forceTLS = false
	}
}

// WithDebug enables debug mode
func WithDebug() Option {
	return func(c *Client) {
		c.debug = true
	}
}

// SendEmail sends an email
func (c *Client) SendEmail(ctx context.Context, email Email) error {
	if err := c.validateEmail(email); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	msg, err := c.buildMessage(email)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	recipients := append(append(email.To, email.CC...), email.BCC...)

	var lastErr error
	for i := 0; i <= c.retryCount; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(i)):
			}
		}

		if err := c.send(ctx, email.From, recipients, msg); err != nil {
			lastErr = err
			if c.debug {
				fmt.Printf("Attempt %d failed: %v\n", i+1, err)
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", c.retryCount+1, lastErr)
}

func (c *Client) send(ctx context.Context, from string, to []string, msg []byte) error {
	addr := net.JoinHostPort(c.host, fmt.Sprintf("%d", c.port))

	conn, client, err := c.establishConnection(ctx, addr)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil && c.debug {
			fmt.Printf("Warning: failed to close SMTP client: %v\n", closeErr)
		}
		if closeErr := conn.Close(); closeErr != nil && c.debug {
			fmt.Printf("Warning: failed to close connection: %v\n", closeErr)
		}
	}()

	if err := c.authenticateClient(client); err != nil {
		return err
	}

	return c.sendMessage(client, from, to, msg)
}

func (c *Client) establishConnection(ctx context.Context, addr string) (net.Conn, *smtp.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil && c.debug {
			fmt.Printf("Warning: failed to set deadline: %v\n", err)
		}
	}

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil && c.debug {
			fmt.Printf("Warning: failed to close connection: %v\n", closeErr)
		}
		return nil, nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if c.forceTLS {
		conn, client, err = c.setupTLS(conn, client, addr)
		if err != nil {
			return nil, nil, err
		}
	}

	return conn, client, nil
}

func (c *Client) setupTLS(conn net.Conn, client *smtp.Client, addr string) (net.Conn, *smtp.Client, error) {
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(c.tlsConfig); err != nil {
			return nil, nil, fmt.Errorf("STARTTLS failed: %w", err)
		}
		return conn, client, nil
	}

	if c.port == 465 {
		if closeErr := conn.Close(); closeErr != nil && c.debug {
			fmt.Printf("Warning: failed to close connection: %v\n", closeErr)
		}
		tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: c.timeout}, "tcp", addr, c.tlsConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("TLS connection failed: %w", err)
		}
		newClient, err := smtp.NewClient(tlsConn, c.host)
		if err != nil {
			if closeErr := tlsConn.Close(); closeErr != nil && c.debug {
				fmt.Printf("Warning: failed to close TLS connection: %v\n", closeErr)
			}
			return nil, nil, fmt.Errorf("failed to create SMTP client over TLS: %w", err)
		}
		return tlsConn, newClient, nil
	}

	return conn, client, nil
}

func (c *Client) authenticateClient(client *smtp.Client) error {
	if c.authType != authNone && c.username != "" {
		auth := c.getAuth()
		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
		}
	}
	return nil
}

func (c *Client) sendMessage(client *smtp.Client, from string, to []string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", addr, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		if closeErr := w.Close(); closeErr != nil && c.debug {
			fmt.Printf("Warning: failed to close data writer: %v\n", closeErr)
		}
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

func (c *Client) getAuth() smtp.Auth {
	switch c.authType {
	case authLogin:
		return &loginAuth{username: c.username, password: c.password}
	case authCRAMMD5:
		return smtp.CRAMMD5Auth(c.username, c.password)
	case authPlain:
		return smtp.PlainAuth("", c.username, c.password, c.host)
	default:
		return nil
	}
}

func (c *Client) validateEmail(email Email) error {
	if email.From == "" {
		return errors.New("from address is required")
	}
	if _, err := mail.ParseAddress(email.From); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	if len(email.To) == 0 && len(email.CC) == 0 && len(email.BCC) == 0 {
		return errors.New("at least one recipient is required")
	}

	for _, addr := range append(append(email.To, email.CC...), email.BCC...) {
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("invalid email address %s: %w", addr, err)
		}
	}

	if email.Subject == "" {
		return errors.New("subject is required")
	}

	if email.HTMLBody == "" && email.TextBody == "" && len(email.Attachments) == 0 {
		return errors.New("email body or attachments required")
	}

	return nil
}

func (c *Client) buildMessage(email Email) ([]byte, error) {
	buf := new(bytes.Buffer)

	headers := c.buildHeaders(email)
	writer := multipart.NewWriter(buf)
	headers.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary()))

	c.writeHeaders(buf, headers)

	if err := c.addMessageBody(writer, email); err != nil {
		return nil, err
	}

	for _, att := range email.Attachments {
		if err := c.addAttachment(writer, att); err != nil {
			return nil, fmt.Errorf("failed to add attachment: %w", err)
		}
	}

	if closeErr := writer.Close(); closeErr != nil && c.debug {
		fmt.Printf("Warning: failed to close multipart writer: %v\n", closeErr)
	}

	return buf.Bytes(), nil
}

func (c *Client) buildHeaders(email Email) textproto.MIMEHeader {
	headers := make(textproto.MIMEHeader)
	headers.Set("From", email.From)
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", email.Subject))
	headers.Set("MIME-Version", "1.0")
	headers.Set("Date", time.Now().Format(time.RFC1123Z))
	headers.Set("Message-ID", generateMessageID(email.From))

	if len(email.To) > 0 {
		headers.Set("To", strings.Join(email.To, ", "))
	}
	if len(email.CC) > 0 {
		headers.Set("Cc", strings.Join(email.CC, ", "))
	}

	for k, v := range email.Headers {
		headers.Set(k, v)
	}

	return headers
}

func (c *Client) writeHeaders(buf *bytes.Buffer, headers textproto.MIMEHeader) {
	for k, v := range headers {
		fmt.Fprintf(buf, "%s: %s\r\n", k, strings.Join(v, ", "))
	}
	fmt.Fprintf(buf, "\r\n")
}

func (c *Client) addMessageBody(writer *multipart.Writer, email Email) error {
	if email.TextBody == "" && email.HTMLBody == "" {
		return nil
	}

	altHeader := textproto.MIMEHeader{}
	altWriter := multipart.NewWriter(&bytes.Buffer{})
	altHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=%s", altWriter.Boundary()))

	pw, err := writer.CreatePart(altHeader)
	if err != nil {
		return err
	}

	altWriter = multipart.NewWriter(pw)

	c.addTextPart(altWriter, email.TextBody)
	c.addHTMLPart(altWriter, email.HTMLBody)

	if closeErr := altWriter.Close(); closeErr != nil && c.debug {
		fmt.Printf("Warning: failed to close alternative writer: %v\n", closeErr)
	}

	return nil
}

func (c *Client) addTextPart(altWriter *multipart.Writer, textBody string) {
	if textBody == "" {
		return
	}

	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	if tw, err := altWriter.CreatePart(textHeader); err == nil {
		if _, writeErr := tw.Write([]byte(textBody)); writeErr != nil && c.debug {
			fmt.Printf("Warning: failed to write text body: %v\n", writeErr)
		}
	}
}

func (c *Client) addHTMLPart(altWriter *multipart.Writer, htmlBody string) {
	if htmlBody == "" {
		return
	}

	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=utf-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	if hw, err := altWriter.CreatePart(htmlHeader); err == nil {
		if _, writeErr := hw.Write([]byte(htmlBody)); writeErr != nil && c.debug {
			fmt.Printf("Warning: failed to write HTML body: %v\n", writeErr)
		}
	}
}

func (c *Client) addAttachment(writer *multipart.Writer, att Attachment) error {
	header := textproto.MIMEHeader{}

	contentType := "application/octet-stream"
	if ext := filepath.Ext(att.Filename); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			contentType = ct
		}
	}

	header.Set("Content-Type", contentType)
	header.Set("Content-Transfer-Encoding", "base64")

	if att.Inline && att.ContentID != "" {
		header.Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.Filename))
		header.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
	} else {
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))
	}

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}

	encoder := base64.NewEncoder(base64.StdEncoding, part)
	defer func() {
		if closeErr := encoder.Close(); closeErr != nil && c.debug {
			fmt.Printf("Warning: failed to close encoder: %v\n", closeErr)
		}
	}()

	if _, err := io.Copy(encoder, att.Content); err != nil {
		return err
	}

	return nil
}

func AttachFile(path string) (Attachment, error) {
	file, err := os.Open(path)
	if err != nil {
		return Attachment{}, err
	}

	return Attachment{
		Filename: filepath.Base(path),
		Content:  file,
		Inline:   false,
	}, nil
}

func generateMessageID(from string) string {
	domain := "localhost"
	if parts := strings.Split(from, "@"); len(parts) == 2 {
		domain = parts[1]
	}
	return fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), os.Getpid(), domain)
}

type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		}
	}
	return nil, nil
}
