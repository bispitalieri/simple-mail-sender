package smtp

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"fmt"
	"net/textproto"
	"strings"
)

func NewAuthenticator(cfg AuthConfig) Authenticator {
	switch cfg.Mechanism {
	case AuthMechanismLOGIN:
		return newLoginAuth(cfg.Username, cfg.Password, "")
	case AuthMechanismCRAMMD5:
		return newCRAMMD5Auth(cfg.Username, cfg.Password)
	case AuthMechanismOAUTH2:
		return newOAuth2Auth(cfg.Username, cfg.Token)
	case AuthMechanismPLAIN:
		fallthrough
	default:
		return newPlainAuth(cfg.Identity, cfg.Username, cfg.Password, "")
	}
}

type plainAuth struct {
	identity, username, password, host string
}

func newPlainAuth(identity, username, password, host string) *plainAuth {
	return &plainAuth{
		identity: identity,
		username: username,
		password: password,
		host:     host,
	}
}

func (a *plainAuth) Authenticate(t *textproto.Conn, mechanism string, cfg AuthConfig) error {
	if mechanism != AuthMechanismPLAIN {
		return fmt.Errorf("plain auth does not support mechanism: %s", mechanism)
	}

	authStr := a.username + "\x00" + a.identity + "\x00" + a.password
	encoded := encodeBase64([]byte(authStr))

	if err := t.PrintfLine("AUTH PLAIN %s", encoded); err != nil {
		return err
	}

	_, _, err := t.ReadResponse(235)
	if err != nil {
		return fmt.Errorf("AUTH PLAIN rejected: %w", err)
	}

	return nil
}

type loginAuth struct {
	username, password, host string
}

func newLoginAuth(username, password, host string) *loginAuth {
	return &loginAuth{
		username: username,
		password: password,
		host:     host,
	}
}

func (a *loginAuth) Authenticate(t *textproto.Conn, mechanism string, cfg AuthConfig) error {
	if mechanism != AuthMechanismLOGIN {
		return fmt.Errorf("login auth does not support mechanism: %s", mechanism)
	}

	if err := t.PrintfLine("AUTH LOGIN"); err != nil {
		return err
	}

	_, msg, err := t.ReadResponse(334)
	if err != nil {
		return fmt.Errorf("AUTH LOGIN rejected: %w", err)
	}
	if !strings.Contains(msg, "VXNlcm5hbWU6") {
		return fmt.Errorf("unexpected AUTH LOGIN response: %s", msg)
	}

	if err := t.PrintfLine("%s", encodeBase64([]byte(a.username))); err != nil {
		return err
	}

	_, msg, err = t.ReadResponse(334)
	if err != nil {
		return fmt.Errorf("AUTH LOGIN username rejected: %w", err)
	}
	if !strings.Contains(msg, "UGFzc3dvcmQ6") {
		return fmt.Errorf("unexpected AUTH LOGIN password challenge: %s", msg)
	}

	if err := t.PrintfLine("%s", encodeBase64([]byte(a.password))); err != nil {
		return err
	}

	_, _, err = t.ReadResponse(235)
	if err != nil {
		return fmt.Errorf("AUTH LOGIN password rejected: %w", err)
	}

	return nil
}

type cramMD5Auth struct {
	username, password string
}

func newCRAMMD5Auth(username, password string) *cramMD5Auth {
	return &cramMD5Auth{
		username: username,
		password: password,
	}
}

func (a *cramMD5Auth) Authenticate(t *textproto.Conn, mechanism string, cfg AuthConfig) error {
	if mechanism != AuthMechanismCRAMMD5 {
		return fmt.Errorf("CRAM-MD5 auth does not support mechanism: %s", mechanism)
	}

	if err := t.PrintfLine("AUTH CRAM-MD5"); err != nil {
		return err
	}

	_, msg, err := t.ReadResponse(334)
	if err != nil {
		return fmt.Errorf("AUTH CRAM-MD5 rejected: %w", err)
	}

	challenge, err := decodeBase64(strings.TrimSpace(msg))
	if err != nil {
		return fmt.Errorf("invalid CRAM-MD5 challenge: %w", err)
	}

	mac := hmac.New(md5.New, []byte(a.password))
	mac.Write(challenge)
	digest := mac.Sum(nil)

	response := fmt.Sprintf("%s %x", a.username, digest)
	encoded := encodeBase64([]byte(response))

	if err := t.PrintfLine("%s", encoded); err != nil {
		return err
	}

	_, _, err = t.ReadResponse(235)
	if err != nil {
		return fmt.Errorf("AUTH CRAM-MD5 authentication failed: %w", err)
	}

	return nil
}

type oauth2Auth struct {
	username, token string
}

func newOAuth2Auth(username, token string) *oauth2Auth {
	return &oauth2Auth{
		username: username,
		token:    token,
	}
}

func (a *oauth2Auth) Authenticate(t *textproto.Conn, mechanism string, cfg AuthConfig) error {
	if mechanism != AuthMechanismOAUTH2 {
		return fmt.Errorf("OAuth2 auth does not support mechanism: %s", mechanism)
	}

	oauth2Str := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.username, a.token)
	encoded := encodeBase64([]byte(oauth2Str))

	if err := t.PrintfLine("AUTH XOAUTH2 %s", encoded); err != nil {
		return err
	}

	_, msg, err := t.ReadResponse(235)
	if err != nil {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(msg)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "SERVFAIL") || strings.Contains(line, "Invalid") {
				return fmt.Errorf("OAuth2 authentication failed: %w", err)
			}
		}
		return fmt.Errorf("AUTH XOAUTH2 rejected: %w", err)
	}

	return nil
}
