package smtp

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				Host: "smtp.example.com",
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismPLAIN,
					Username:  "user@example.com",
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode: TLSModeSTARTTLS,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: false,
		},
		{
			name: "missing host",
			config: Config{
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismPLAIN,
					Username:  "user@example.com",
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode: TLSModeSTARTTLS,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "smtp host is required",
		},
		{
			name: "invalid port",
			config: Config{
				Host: "smtp.example.com",
				Port: 0,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismPLAIN,
					Username:  "user@example.com",
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode: TLSModeSTARTTLS,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "smtp port must be between 1 and 65535",
		},
		{
			name: "unsupported TLS mode",
			config: Config{
				Host: "smtp.example.com",
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismPLAIN,
					Username:  "user@example.com",
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode: "invalid",
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "unsupported tls mode",
		},
		{
			name: "unsupported auth mechanism",
			config: Config{
				Host: "smtp.example.com",
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: "INVALID",
					Username:  "user@example.com",
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode: TLSModeSTARTTLS,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "unsupported auth mechanism",
		},
		{
			name: "PLAIN auth without username",
			config: Config{
				Host: "smtp.example.com",
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismPLAIN,
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode: TLSModeSTARTTLS,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "auth username is required",
		},
		{
			name: "OAUTH2 without token",
			config: Config{
				Host: "smtp.example.com",
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismOAUTH2,
					Username:  "user@example.com",
				},
				TLS: TLSConfig{
					Mode: TLSModeSTARTTLS,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "auth token is required",
		},
		{
			name: "invalid connect timeout",
			config: Config{
				Host: "smtp.example.com",
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismPLAIN,
					Username:  "user@example.com",
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode: TLSModeSTARTTLS,
				},
				Timeouts: TimeoutsConfig{
					Connect: 0,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "timeouts.connect must be positive",
		},
		{
			name: "client cert without key",
			config: Config{
				Host: "smtp.example.com",
				Port: 587,
				Auth: AuthConfig{
					Enabled:   true,
					Mechanism: AuthMechanismPLAIN,
					Username:  "user@example.com",
					Password:  "password",
				},
				TLS: TLSConfig{
					Mode:           TLSModeSTARTTLS,
					ClientCertPath: "/path/to/cert.crt",
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: true,
			errMsg:    "tls client_key_path is required",
		},
		{
			name: "auth disabled is valid",
			config: Config{
				Host: "smtp.example.com",
				Port: 25,
				Auth: AuthConfig{
					Enabled: false,
				},
				TLS: TLSConfig{
					Mode: TLSModeNone,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveServerName(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected string
	}{
		{
			name: "custom server name",
			cfg: Config{
				Host: "smtp.example.com",
				TLS: TLSConfig{
					ServerName: "mail.example.com",
				},
			},
			expected: "mail.example.com",
		},
		{
			name: "default server name",
			cfg: Config{
				Host: "smtp.example.com",
				TLS: TLSConfig{
					ServerName: "",
				},
			},
			expected: "smtp.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveServerName(&tt.cfg)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEncodeDecodeBase64(t *testing.T) {
	data := []byte("test data")
	encoded := encodeBase64(data)
	decoded, err := decodeBase64(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Errorf("expected %q, got %q", data, decoded)
	}
}

func TestMessageBytes(t *testing.T) {
	msg := NewMessage("sender@example.com", "Test Subject", "Test Body", "recipient@example.com")
	msg.SetHeader("X-Custom", "value")

	data := msg.Bytes()

	if !strings.Contains(string(data), "From: sender@example.com\r\n") {
		t.Error("From header missing")
	}
	if !strings.Contains(string(data), "To: recipient@example.com\r\n") {
		t.Error("To header missing")
	}
	if !strings.Contains(string(data), "Subject: Test Subject\r\n") {
		t.Error("Subject header missing")
	}
	if !strings.Contains(string(data), "X-Custom: value\r\n") {
		t.Error("Custom header missing")
	}
	if !strings.Contains(string(data), "\r\n\r\nTest Body\r\n") {
		t.Error("Body missing")
	}
}

func TestAuthMechanisms(t *testing.T) {
	t.Run("PLAIN auth encoding", func(t *testing.T) {
		auth := newPlainAuth("", "user", "pass", "example.com")
		if auth == nil {
			t.Fatal("expected non-nil auth")
		}
	})

	t.Run("LOGIN auth encoding", func(t *testing.T) {
		auth := newLoginAuth("user", "pass", "example.com")
		if auth == nil {
			t.Fatal("expected non-nil auth")
		}
	})

	t.Run("CRAM-MD5 auth encoding", func(t *testing.T) {
		auth := newCRAMMD5Auth("user", "pass")
		if auth == nil {
			t.Fatal("expected non-nil auth")
		}
	})

	t.Run("OAUTH2 auth encoding", func(t *testing.T) {
		auth := newOAuth2Auth("user", "token")
		if auth == nil {
			t.Fatal("expected non-nil auth")
		}
	})
}

func startMockSMTPServer(t *testing.T) (net.Listener, string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleMockSMTP(conn)
		}
	}()

	return listener, listener.Addr().String()
}

func handleMockSMTP(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	greeting := "220 mock.smtp.server ESMTP\r\n"
	conn.Write([]byte(greeting))

	pendingMechanism := ""
	dataMode := false
	var dataBuf bytes.Buffer

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		response := "500 Unknown command\r\n"

		if dataMode {
			if line == "." {
				dataMode = false
				response = "250 OK: queued\r\n"
			} else {
				dataBuf.WriteString(line)
				dataBuf.WriteString("\r\n")
				continue
			}
		} else if pendingMechanism == "CRAM-MD5" {
			response = "235 Authentication successful\r\n"
			pendingMechanism = ""
		} else if pendingMechanism == "LOGIN_USER" {
			response = "334 UGFzc3dvcmQ6\r\n"
			pendingMechanism = "LOGIN_PASS"
		} else if pendingMechanism == "LOGIN_PASS" {
			response = "235 Authentication successful\r\n"
			pendingMechanism = ""
		} else {
			if strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO") {
				response = "250 mock.smtp.server\r\n"
			} else if line == "STARTTLS" {
				response = "220 Ready to start TLS\r\n"
			} else if strings.HasPrefix(line, "AUTH PLAIN") {
				response = "235 Authentication successful\r\n"
			} else if strings.HasPrefix(line, "AUTH LOGIN") {
				response = "334 VXNlcm5hbWU6\r\n"
				pendingMechanism = "LOGIN_USER"
			} else if strings.HasPrefix(line, "AUTH CRAM-MD5") {
				challenge := encodeBase64([]byte("mockchallenge"))
				response = "334 " + challenge + "\r\n"
				pendingMechanism = "CRAM-MD5"
			} else if strings.HasPrefix(line, "AUTH XOAUTH2") {
				response = "235 Authentication successful\r\n"
			} else if strings.HasPrefix(line, "MAIL FROM:") {
				response = "250 OK\r\n"
			} else if strings.HasPrefix(line, "RCPT TO:") {
				response = "250 OK\r\n"
			} else if line == "DATA" {
				dataMode = true
				dataBuf.Reset()
				response = "354 End data with <CR><LF>.<CR><LF>\r\n"
			} else if line == "QUIT" {
				response = "221 Bye\r\n"
			}
		}

		conn.Write([]byte(response))
	}
}

func TestSMTPClientConnection(t *testing.T) {
	_, addr := startMockSMTPServer(t)

	cfg := &Config{
		Host:      strings.Split(addr, ":")[0],
		Port:      mustParsePort(addr),
		LocalName: "localhost",
		Auth: AuthConfig{
			Enabled:   true,
			Mechanism: AuthMechanismPLAIN,
			Username:  "user@example.com",
			Password:  "password",
		},
		TLS: TLSConfig{
			Mode: TLSModeNone,
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	auth := NewAuthenticator(cfg.Auth)
	if err := client.Auth(auth); err != nil {
		t.Fatalf("auth failed: %v", err)
	}

	msg := NewMessage("sender@example.com", "Test", "Body", "recipient@example.com")
	if err := client.Send(cfg.Defaults.FromAddress, []string{"recipient@example.com"}, msg.Bytes()); err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

func TestSMTPClientNoAuth(t *testing.T) {
	_, addr := startMockSMTPServer(t)

	cfg := &Config{
		Host:      strings.Split(addr, ":")[0],
		Port:      mustParsePort(addr),
		LocalName: "localhost",
		Auth: AuthConfig{
			Enabled: false,
		},
		TLS: TLSConfig{
			Mode: TLSModeNone,
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	msg := NewMessage("sender@example.com", "Test", "Body", "recipient@example.com")
	if err := client.Send(cfg.Defaults.FromAddress, []string{"recipient@example.com"}, msg.Bytes()); err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

func TestSMTPClientSTARTTLS(t *testing.T) {
	listener, addr := startMockTLSSMTPServer(t)
	defer listener.Close()

	cfg := &Config{
		Host:      strings.Split(addr, ":")[0],
		Port:      mustParsePort(addr),
		LocalName: "localhost",
		Auth: AuthConfig{
			Enabled: false,
		},
		TLS: TLSConfig{
			Mode:       TLSModeSTARTTLS,
			SkipVerify: true,
			ServerName: strings.Split(addr, ":")[0],
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect with STARTTLS: %v", err)
	}
	defer client.Close()

	msg := NewMessage("sender@example.com", "Test", "Body", "recipient@example.com")
	if err := client.Send(cfg.Defaults.FromAddress, []string{"recipient@example.com"}, msg.Bytes()); err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

func startMockTLSSMTPServer(t *testing.T) (net.Listener, string) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start TLS server: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleMockSMTPWithSTARTTLS(conn, cert)
		}
	}()

	return listener, listener.Addr().String()
}

func handleMockSMTPWithSTARTTLS(conn net.Conn, cert tls.Certificate) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	greeting := "220 mock.smtp.server ESMTP\r\n"
	conn.Write([]byte(greeting))

	tlsUpgraded := false
	dataMode := false
	var dataBuf bytes.Buffer

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		response := "500 Unknown command\r\n"

		if tlsUpgraded {
			if dataMode {
				if line == "." {
					dataMode = false
					response = "250 OK: queued\r\n"
				} else {
					dataBuf.WriteString(line)
					dataBuf.WriteString("\r\n")
					continue
				}
			} else {
				if strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO") {
					response = "250-STARTTLS\r\n250 OK\r\n"
				} else if strings.HasPrefix(line, "MAIL FROM:") {
					response = "250 OK\r\n"
				} else if strings.HasPrefix(line, "RCPT TO:") {
					response = "250 OK\r\n"
				} else if line == "DATA" {
					dataMode = true
					dataBuf.Reset()
					response = "354 End data with <CR><LF>.<CR><LF>\r\n"
				} else if line == "QUIT" {
					response = "221 Bye\r\n"
				}
			}
		} else {
			if strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO") {
				response = "250-STARTTLS\r\n250 OK\r\n"
			} else if line == "STARTTLS" {
				response = "220 Ready to start TLS\r\n"
				conn.Write([]byte(response))

				tlsConn := tls.Server(conn, &tls.Config{
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true,
				})
				if err := tlsConn.Handshake(); err != nil {
					return
				}

				conn = tlsConn
				reader = bufio.NewReader(conn)
				tlsUpgraded = true
				continue
			} else if strings.HasPrefix(line, "QUIT") {
				response = "221 Bye\r\n"
			}
		}

		conn.Write([]byte(response))
	}
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func mustParsePort(addr string) int {
	_, portStr, _ := net.SplitHostPort(addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}

func TestConfigAddress(t *testing.T) {
	cfg := Config{
		Host: "smtp.example.com",
		Port: 587,
	}
	expected := "smtp.example.com:587"
	if cfg.Address() != expected {
		t.Errorf("expected %q, got %q", expected, cfg.Address())
	}
}

func TestConfigDefaultFrom(t *testing.T) {
	cfg := Config{
		Host: "smtp.example.com",
		Port: 587,
		Defaults: DefaultsConfig{
			FromAddress: "noreply@example.com",
			FromName:    "Test App",
		},
	}

	if cfg.Defaults.FromAddress != "noreply@example.com" {
		t.Error("FromAddress mismatch")
	}
	if cfg.Defaults.FromName != "Test App" {
		t.Error("FromName mismatch")
	}
}

func TestAuthDisabled(t *testing.T) {
	_, addr := startMockSMTPServer(t)

	cfg := &Config{
		Host: strings.Split(addr, ":")[0],
		Port: mustParsePort(addr),
		Auth: AuthConfig{
			Enabled: false,
		},
		TLS: TLSConfig{
			Mode: TLSModeNone,
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	if err := client.Auth(nil); err != nil {
		t.Fatalf("auth with nil should be skipped: %v", err)
	}
}

func TestUnsupportedTLSMode(t *testing.T) {
	cfg := &Config{
		Host: "smtp.example.com",
		Port: 587,
		TLS: TLSConfig{
			Mode: "invalid",
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
	}

	_, err := NewSMTPClient(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported TLS mode")
	}
	if !strings.Contains(err.Error(), "unsupported tls mode") {
		t.Errorf("expected unsupported tls mode error, got: %v", err)
	}
}

func TestSTARTTLSNotSupported(t *testing.T) {
	// Start a plain server that doesn't support STARTTLS
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				c.Write([]byte("220 plain.server ESMTP\r\n"))
				buf := make([]byte, 1024)
				for {
					n, _ := c.Read(buf)
					if n == 0 {
						return
					}
					line := strings.TrimSpace(string(buf[:n]))
					if strings.HasPrefix(line, "EHLO") {
						c.Write([]byte("250 plain.server\r\n"))
					} else if line == "STARTTLS" {
						c.Write([]byte("500 Command not recognized\r\n"))
					} else if line == "QUIT" {
						c.Write([]byte("221 Bye\r\n"))
					} else {
						c.Write([]byte("500 Command not recognized\r\n"))
					}
				}
			}(conn)
		}
	}()

	addr := listener.Addr().String()
	cfg := &Config{
		Host: strings.Split(addr, ":")[0],
		Port: mustParsePort(addr),
		TLS: TLSConfig{
			Mode: TLSModeSTARTTLS,
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	_, err = NewSMTPClient(cfg)
	if err == nil {
		t.Fatal("expected error when STARTTLS not supported")
	}
	if !strings.Contains(err.Error(), "STARTTLS not supported") {
		t.Errorf("expected STARTTLS not supported error, got: %v", err)
	}
}

func TestConnectionTimeout(t *testing.T) {
	// Try to connect to a non-routable address with a short timeout
	cfg := &Config{
		Host: "192.0.2.1",
		Port: 25,
		TLS: TLSConfig{
			Mode: TLSModeNone,
		},
		Timeouts: TimeoutsConfig{
			Connect: 1 * time.Millisecond,
			Send:    5 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	start := time.Now()
	_, err := NewSMTPClient(cfg)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected connection timeout error")
	}
	if elapsed > 5*time.Second {
		t.Errorf("connection took too long: %v", elapsed)
	}
}

func TestMessageFormatting(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		contains []string
	}{
		{
			name: "basic message",
			msg:  NewMessage("from@example.com", "Subject", "Body", "to@example.com"),
			contains: []string{
				"From: from@example.com\r\n",
				"To: to@example.com\r\n",
				"Subject: Subject\r\n",
				"\r\nBody\r\n",
			},
		},
		{
			name: "message with display name",
			msg: func() *Message {
				msg := NewMessage("from@example.com", "Subject", "Body", "to@example.com")
				msg.SetHeader("From", "Display Name <from@example.com>")
				return msg
			}(),
			contains: []string{
				"From: Display Name <from@example.com>\r\n",
				"Subject: Subject\r\n",
			},
		},
		{
			name: "message with custom headers",
			msg: func() *Message {
				msg := NewMessage("from@example.com", "Subject", "Body", "to@example.com")
				msg.SetHeader("X-Priority", "1")
				msg.SetHeader("X-Custom", "value")
				return msg
			}(),
			contains: []string{
				"X-Priority: 1\r\n",
				"X-Custom: value\r\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.msg.Bytes()
			for _, substr := range tt.contains {
				if !strings.Contains(string(data), substr) {
					t.Errorf("expected message to contain %q", substr)
				}
			}
		})
	}
}

func TestCRAMMD5Auth(t *testing.T) {
	_, addr := startMockSMTPServer(t)

	cfg := &Config{
		Host: strings.Split(addr, ":")[0],
		Port: mustParsePort(addr),
		Auth: AuthConfig{
			Enabled:   true,
			Mechanism: AuthMechanismCRAMMD5,
			Username:  "user@example.com",
			Password:  "password",
		},
		TLS: TLSConfig{
			Mode: TLSModeNone,
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	auth := NewAuthenticator(cfg.Auth)
	if err := client.Auth(auth); err != nil {
		t.Fatalf("CRAM-MD5 auth failed: %v", err)
	}
}

func TestLOGINAuth(t *testing.T) {
	_, addr := startMockSMTPServer(t)

	cfg := &Config{
		Host: strings.Split(addr, ":")[0],
		Port: mustParsePort(addr),
		Auth: AuthConfig{
			Enabled:   true,
			Mechanism: AuthMechanismLOGIN,
			Username:  "user@example.com",
			Password:  "password",
		},
		TLS: TLSConfig{
			Mode: TLSModeNone,
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	auth := NewAuthenticator(cfg.Auth)
	if err := client.Auth(auth); err != nil {
		t.Fatalf("LOGIN auth failed: %v", err)
	}
}

func TestOAuth2Auth(t *testing.T) {
	_, addr := startMockSMTPServer(t)

	cfg := &Config{
		Host: strings.Split(addr, ":")[0],
		Port: mustParsePort(addr),
		Auth: AuthConfig{
			Enabled:   true,
			Mechanism: AuthMechanismOAUTH2,
			Username:  "user@example.com",
			Token:     "oauth_token",
		},
		TLS: TLSConfig{
			Mode: TLSModeNone,
		},
		Timeouts: TimeoutsConfig{
			Connect: 5 * time.Second,
			Send:    10 * time.Second,
		},
		Defaults: DefaultsConfig{
			FromAddress: "sender@example.com",
		},
	}

	client, err := NewSMTPClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	auth := NewAuthenticator(cfg.Auth)
	if err := client.Auth(auth); err != nil {
		t.Fatalf("OAUTH2 auth failed: %v", err)
	}
}

func TestCACertLoading(t *testing.T) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})

	certPool, err := createCertPool(certPEM)
	if err != nil {
		t.Fatalf("failed to create cert pool: %v", err)
	}

	if certPool == nil {
		t.Fatal("expected non-nil cert pool")
	}
}

func TestLoadClientCert(t *testing.T) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey))})

	tmpDir := t.TempDir()
	certPath := tmpDir + "/cert.pem"
	keyPath := tmpDir + "/key.pem"

	os.WriteFile(certPath, certPEM, 0644)
	os.WriteFile(keyPath, keyPEM, 0644)

	tlsCfg, err := loadClientCert(certPath, keyPath)
	if err != nil {
		t.Fatalf("failed to load client cert: %v", err)
	}

	if len(tlsCfg.Certificates) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(tlsCfg.Certificates))
	}
}

func TestTLSConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
		errMsg    string
	}{
		{
			name: "valid SSL TLS config",
			cfg: Config{
				Host: "smtp.example.com",
				Port: 465,
				TLS: TLSConfig{
					Mode: TLSModeSSL,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: false,
		},
		{
			name: "valid no TLS config",
			cfg: Config{
				Host: "smtp.example.com",
				Port: 25,
				TLS: TLSConfig{
					Mode: TLSModeNone,
				},
				Timeouts: TimeoutsConfig{
					Connect: 10 * time.Second,
					Send:    30 * time.Second,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		Host: "smtp.example.com",
		Port: 587,
		Defaults: DefaultsConfig{
			FromAddress: "noreply@example.com",
			FromName:    "Test App",
		},
		Timeouts: TimeoutsConfig{
			Connect: 10 * time.Second,
			Send:    30 * time.Second,
		},
		TLS: TLSConfig{
			Mode: TLSModeSTARTTLS,
		},
	}

	if cfg.Defaults.FromAddress != "noreply@example.com" {
		t.Errorf("expected FromAddress to be noreply@example.com, got %s", cfg.Defaults.FromAddress)
	}
	if cfg.Defaults.FromName != "Test App" {
		t.Errorf("expected FromName to be Test App, got %s", cfg.Defaults.FromName)
	}
}

func TestAuthMechanismStrings(t *testing.T) {
	if AuthMechanismPLAIN != "PLAIN" {
		t.Errorf("expected PLAIN, got %s", AuthMechanismPLAIN)
	}
	if AuthMechanismLOGIN != "LOGIN" {
		t.Errorf("expected LOGIN, got %s", AuthMechanismLOGIN)
	}
	if AuthMechanismCRAMMD5 != "CRAM-MD5" {
		t.Errorf("expected CRAM-MD5, got %s", AuthMechanismCRAMMD5)
	}
	if AuthMechanismOAUTH2 != "OAUTH2" {
		t.Errorf("expected OAUTH2, got %s", AuthMechanismOAUTH2)
	}
}

func TestTLSModeStrings(t *testing.T) {
	if TLSModeNone != "none" {
		t.Errorf("expected none, got %s", TLSModeNone)
	}
	if TLSModeSTARTTLS != "starttls" {
		t.Errorf("expected starttls, got %s", TLSModeSTARTTLS)
	}
	if TLSModeSSL != "ssl_tls" {
		t.Errorf("expected ssl_tls, got %s", TLSModeSSL)
	}
}

func TestConfigAddressMethod(t *testing.T) {
	tests := []struct {
		host     string
		port     int
		expected string
	}{
		{"smtp.example.com", 25, "smtp.example.com:25"},
		{"smtp.example.com", 587, "smtp.example.com:587"},
		{"smtp.example.com", 465, "smtp.example.com:465"},
		{"localhost", 1025, "localhost:1025"},
	}

	for _, tt := range tests {
		cfg := Config{Host: tt.host, Port: tt.port}
		result := cfg.Address()
		if result != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, result)
		}
	}
}

func TestInvalidDurationConfig(t *testing.T) {
	cfg := Config{
		Host: "smtp.example.com",
		Port: 587,
		TLS: TLSConfig{
			Mode: TLSModeSTARTTLS,
		},
		Timeouts: TimeoutsConfig{
			Connect: -1 * time.Second,
			Send:    30 * time.Second,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative connect timeout")
	}
	if !strings.Contains(err.Error(), "timeouts.connect must be positive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMissingCertificateFiles(t *testing.T) {
	cfg := Config{
		Host: "smtp.example.com",
		Port: 587,
		TLS: TLSConfig{
			Mode:       TLSModeSTARTTLS,
			CACertPath: "/nonexistent/ca.crt",
		},
		Timeouts: TimeoutsConfig{
			Connect: 10 * time.Second,
			Send:    30 * time.Second,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing CA cert file")
	}
	if !strings.Contains(err.Error(), "ca_cert_path does not exist") {
		t.Errorf("unexpected error message: %v", err)
	}
}
