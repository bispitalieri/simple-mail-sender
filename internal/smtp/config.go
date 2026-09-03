package smtp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type TLSConfig struct {
	Mode           string `yaml:"mode" json:"mode"`
	SkipVerify     bool   `yaml:"skip_verify" json:"skip_verify"`
	ServerName     string `yaml:"server_name" json:"server_name"`
	CACertPath     string `yaml:"ca_cert_path" json:"ca_cert_path"`
	ClientCertPath string `yaml:"client_cert_path" json:"client_cert_path"`
	ClientKeyPath  string `yaml:"client_key_path" json:"client_key_path"`
}

type AuthConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Mechanism string `yaml:"mechanism" json:"mechanism"`
	Username  string `yaml:"username" json:"username"`
	Password  string `yaml:"password" json:"password"`
	Token     string `yaml:"token" json:"token"`
	Identity  string `yaml:"identity" json:"identity"`
}

type TimeoutsConfig struct {
	Connect time.Duration `yaml:"connect" json:"connect"`
	Send    time.Duration `yaml:"send" json:"send"`
}

type DefaultsConfig struct {
	FromAddress string `yaml:"from_address" json:"from_address"`
	FromName    string `yaml:"from_name" json:"from_name"`
}

type Config struct {
	Host      string         `yaml:"host" json:"host"`
	Port      int            `yaml:"port" json:"port"`
	LocalName string         `yaml:"local_name" json:"local_name"`
	Auth      AuthConfig     `yaml:"auth" json:"auth"`
	TLS       TLSConfig      `yaml:"tls" json:"tls"`
	Timeouts  TimeoutsConfig `yaml:"timeouts" json:"timeouts"`
	Defaults  DefaultsConfig `yaml:"defaults" json:"defaults"`
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("smtp host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("smtp port must be between 1 and 65535, got: %d", c.Port)
	}

	validTLSModes := map[string]bool{
		TLSModeNone:     true,
		TLSModeSTARTTLS: true,
		TLSModeSSL:      true,
	}
	if !validTLSModes[c.TLS.Mode] {
		return fmt.Errorf("unsupported tls mode: %s (supported: none, starttls, ssl_tls)", c.TLS.Mode)
	}

	if c.TLS.ClientCertPath != "" && c.TLS.ClientKeyPath == "" {
		return errors.New("tls client_key_path is required when client_cert_path is provided")
	}
	if c.TLS.ClientKeyPath != "" && c.TLS.ClientCertPath == "" {
		return errors.New("tls client_cert_path is required when client_key_path is provided")
	}

	if c.Auth.Enabled {
		validMechanisms := map[string]bool{
			AuthMechanismPLAIN:   true,
			AuthMechanismLOGIN:   true,
			AuthMechanismCRAMMD5: true,
			AuthMechanismOAUTH2:  true,
		}
		if !validMechanisms[c.Auth.Mechanism] {
			return fmt.Errorf("unsupported auth mechanism: %s (supported: PLAIN, LOGIN, CRAM-MD5, OAUTH2)", c.Auth.Mechanism)
		}

		switch c.Auth.Mechanism {
		case AuthMechanismPLAIN, AuthMechanismLOGIN, AuthMechanismCRAMMD5:
			if strings.TrimSpace(c.Auth.Username) == "" {
				return fmt.Errorf("auth username is required for mechanism %s", c.Auth.Mechanism)
			}
			if strings.TrimSpace(c.Auth.Password) == "" {
				return fmt.Errorf("auth password is required for mechanism %s", c.Auth.Mechanism)
			}
		case AuthMechanismOAUTH2:
			if strings.TrimSpace(c.Auth.Username) == "" {
				return errors.New("auth username is required for OAUTH2 mechanism")
			}
			if strings.TrimSpace(c.Auth.Token) == "" {
				return errors.New("auth token is required for OAUTH2 mechanism")
			}
		}
	}

	if c.Timeouts.Connect <= 0 {
		return fmt.Errorf("timeouts.connect must be positive, got: %v", c.Timeouts.Connect)
	}
	if c.Timeouts.Send <= 0 {
		return fmt.Errorf("timeouts.send must be positive, got: %v", c.Timeouts.Send)
	}

	if c.TLS.CACertPath != "" {
		if _, err := os.Stat(c.TLS.CACertPath); os.IsNotExist(err) {
			return fmt.Errorf("tls ca_cert_path does not exist: %s", c.TLS.CACertPath)
		}
	}
	if c.TLS.ClientCertPath != "" {
		if _, err := os.Stat(c.TLS.ClientCertPath); os.IsNotExist(err) {
			return fmt.Errorf("tls client_cert_path does not exist: %s", c.TLS.ClientCertPath)
		}
	}
	if c.TLS.ClientKeyPath != "" {
		if _, err := os.Stat(c.TLS.ClientKeyPath); os.IsNotExist(err) {
			return fmt.Errorf("tls client_key_path does not exist: %s", c.TLS.ClientKeyPath)
		}
	}

	return nil
}

func createCertPool(data []byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(data) {
		return nil, errors.New("failed to append certificates to pool")
	}
	return certPool, nil
}

func loadClientCert(certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

func buildTLSConfig(cfg *Config, serverName string) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: cfg.TLS.SkipVerify,
	}

	if cfg.TLS.CACertPath != "" {
		certPool, err := createCertPoolFromFile(cfg.TLS.CACertPath)
		if err != nil {
			return nil, err
		}
		tlsCfg.RootCAs = certPool
	}

	if cfg.TLS.ClientCertPath != "" && cfg.TLS.ClientKeyPath != "" {
		certConfig, err := loadClientCert(cfg.TLS.ClientCertPath, cfg.TLS.ClientKeyPath)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = certConfig.Certificates
	}

	return tlsCfg, nil
}

func createCertPoolFromFile(caCertPath string) (*x509.CertPool, error) {
	data, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate file %s: %w", caCertPath, err)
	}

	certPool, err := createCertPool(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate file %s: %w", caCertPath, err)
	}

	return certPool, nil
}

func resolveServerName(cfg *Config) string {
	if cfg.TLS.ServerName != "" {
		return cfg.TLS.ServerName
	}
	return cfg.Host
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
