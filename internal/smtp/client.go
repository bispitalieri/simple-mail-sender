package smtp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strings"
)

const (
	TLSModeNone     = "none"
	TLSModeSTARTTLS = "starttls"
	TLSModeSSL      = "ssl_tls"
)

const (
	AuthMechanismPLAIN   = "PLAIN"
	AuthMechanismLOGIN   = "LOGIN"
	AuthMechanismCRAMMD5 = "CRAM-MD5"
	AuthMechanismOAUTH2  = "OAUTH2"
)

type SMTPClient struct {
	config *Config
	conn   net.Conn
	text   *textproto.Conn
}

func NewSMTPClient(cfg *Config) (*SMTPClient, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	serverName := resolveServerName(cfg)

	switch cfg.TLS.Mode {
	case TLSModeSSL:
		return connectImplicitTLS(cfg, serverName)
	case TLSModeNone, TLSModeSTARTTLS:
		return connectPlain(cfg, serverName)
	default:
		return nil, fmt.Errorf("unsupported tls mode: %s", cfg.TLS.Mode)
	}
}

func connectImplicitTLS(cfg *Config, serverName string) (*SMTPClient, error) {
	tlsCfg, err := buildTLSConfig(cfg, serverName)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout: cfg.Timeouts.Connect,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", cfg.Address(), tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect with implicit TLS: %w", err)
	}

	text := textproto.NewConn(conn)
	_, _, err = text.ReadResponse(220)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read SMTP greeting: %w", err)
	}

	if cfg.LocalName != "" {
		if err := text.PrintfLine("EHLO %s", cfg.LocalName); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to send EHLO: %w", err)
		}
		_, _, err = text.ReadResponse(250)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read EHLO response: %w", err)
		}
	}

	return &SMTPClient{
		config: cfg,
		conn:   conn,
		text:   text,
	}, nil
}

func connectPlain(cfg *Config, serverName string) (*SMTPClient, error) {
	dialer := &net.Dialer{
		Timeout: cfg.Timeouts.Connect,
	}

	conn, err := dialer.Dial("tcp", cfg.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	text := textproto.NewConn(conn)
	_, _, err = text.ReadResponse(220)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read SMTP greeting: %w", err)
	}

	localName := cfg.LocalName
	if localName == "" {
		localName = "localhost"
	}

	if err := text.PrintfLine("EHLO %s", localName); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send EHLO: %w", err)
	}

	_, resp, err := text.ReadResponse(250)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read EHLO response: %w", err)
	}

	if cfg.TLS.Mode == TLSModeSTARTTLS {
		if !strings.Contains(resp, "STARTTLS") {
			conn.Close()
			return nil, errors.New("STARTTLS not supported by server")
		}

		if err := text.PrintfLine("STARTTLS"); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to send STARTTLS: %w", err)
		}

		_, _, err = text.ReadResponse(220)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("STARTTLS command failed: %w", err)
		}

		tlsCfg, err := buildTLSConfig(cfg, serverName)
		if err != nil {
			conn.Close()
			return nil, err
		}

		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("TLS handshake failed: %w", err)
		}

		text = textproto.NewConn(tlsConn)
		conn = tlsConn

		if err := text.PrintfLine("EHLO %s", localName); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to send EHLO after STARTTLS: %w", err)
		}
		_, _, err = text.ReadResponse(250)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read EHLO response after STARTTLS: %w", err)
		}
	}

	return &SMTPClient{
		config: cfg,
		conn:   conn,
		text:   text,
	}, nil
}

func (c *SMTPClient) Auth(auth Authenticator) error {
	if !c.config.Auth.Enabled {
		return nil
	}

	mechanism := c.config.Auth.Mechanism
	if mechanism == "" {
		mechanism = AuthMechanismPLAIN
	}

	if err := auth.Authenticate(c.text, mechanism, c.config.Auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

func (c *SMTPClient) Mail(from string) error {
	if err := c.text.PrintfLine("MAIL FROM:<%s>", from); err != nil {
		return err
	}
	_, _, err := c.text.ReadResponse(250)
	return err
}

func (c *SMTPClient) Rcpt(to string) error {
	if err := c.text.PrintfLine("RCPT TO:<%s>", to); err != nil {
		return err
	}
	_, _, err := c.text.ReadResponse(250)
	return err
}

func (c *SMTPClient) Data(data []byte) error {
	if err := c.text.PrintfLine("DATA"); err != nil {
		return err
	}
	_, _, err := c.text.ReadResponse(354)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}
		if err := c.text.PrintfLine("%s", line); err != nil {
			return err
		}
	}

	if err := c.text.PrintfLine("."); err != nil {
		return err
	}

	_, _, err = c.text.ReadResponse(250)
	return err
}

func (c *SMTPClient) Quit() error {
	if c.text != nil {
		_ = c.text.PrintfLine("QUIT")
	}
	return nil
}

func (c *SMTPClient) Close() error {
	var err error
	if c.text != nil {
		_ = c.text.Close()
	}
	if c.conn != nil {
		err = c.conn.Close()
	}
	return err
}

func (c *SMTPClient) Send(from string, to []string, data []byte) error {
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	for _, recipient := range to {
		if err := c.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", recipient, err)
		}
	}

	if err := c.Data(data); err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}

	return nil
}

type Authenticator interface {
	Authenticate(t *textproto.Conn, mechanism string, cfg AuthConfig) error
}

type Message struct {
	From    string
	To      []string
	Subject string
	Body    string
	Headers map[string]string
}

func NewMessage(from, subject, body string, to ...string) *Message {
	return &Message{
		From:    from,
		To:      to,
		Subject: subject,
		Body:    body,
		Headers: make(map[string]string),
	}
}

func (m *Message) Bytes() []byte {
	var buf strings.Builder

	if m.From != "" {
		buf.WriteString("From: ")
		buf.WriteString(m.From)
		buf.WriteString("\r\n")
	}

	for _, to := range m.To {
		buf.WriteString("To: ")
		buf.WriteString(to)
		buf.WriteString("\r\n")
	}

	if m.Subject != "" {
		buf.WriteString("Subject: ")
		buf.WriteString(m.Subject)
		buf.WriteString("\r\n")
	}

	for k, v := range m.Headers {
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(v)
		buf.WriteString("\r\n")
	}

	buf.WriteString("\r\n")
	buf.WriteString(m.Body)
	buf.WriteString("\r\n")

	return []byte(buf.String())
}

func (m *Message) SetHeader(key, value string) {
	m.Headers[key] = value
}
