package auth

import (
	"crypto/tls"
	"fmt"

	ldap "github.com/go-ldap/ldap/v3"
	"github.com/statusy/statusy/config"
)

// LDAPUser holds attributes returned from a successful LDAP authentication.
type LDAPUser struct {
	Username    string
	Email       string
	DisplayName string
}

// AuthenticateLDAP binds to the LDAP server with the user's credentials and
// returns their attributes on success.
func AuthenticateLDAP(username, password string, cfg *config.LDAPConfig) (*LDAPUser, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("LDAP is not enabled")
	}

	conn, err := dialLDAP(cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to LDAP: %w", err)
	}
	defer conn.Close()

	// Bind with service account to search for the user
	if err := conn.Bind(cfg.BindDN, cfg.BindPass); err != nil {
		return nil, fmt.Errorf("LDAP service bind failed: %w", err)
	}

	// Search for the user entry
	filter := fmt.Sprintf(cfg.UserFilter, ldap.EscapeFilter(username))
	searchReq := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, // size limit
		0, // time limit
		false,
		filter,
		[]string{"dn", cfg.AttrEmail, cfg.AttrName},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found in LDAP")
	}

	userDN := result.Entries[0].DN
	email := result.Entries[0].GetAttributeValue(cfg.AttrEmail)
	displayName := result.Entries[0].GetAttributeValue(cfg.AttrName)

	// Bind as the user to verify password
	if err := conn.Bind(userDN, password); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return &LDAPUser{
		Username:    username,
		Email:       email,
		DisplayName: displayName,
	}, nil
}

// TestConnection verifies that the LDAP config is reachable and the service
// account bind succeeds. Used by the admin UI "Test Connection" button.
func TestConnection(cfg *config.LDAPConfig) error {
	conn, err := dialLDAP(cfg)
	if err != nil {
		return fmt.Errorf("connecting to LDAP: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPass); err != nil {
		return fmt.Errorf("service account bind failed: %w", err)
	}

	return nil
}

func dialLDAP(cfg *config.LDAPConfig) (*ldap.Conn, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: false} //nolint:gosec

	switch cfg.TLS {
	case "tls":
		return ldap.DialURL(cfg.URL, ldap.DialWithTLSConfig(tlsCfg))
	case "starttls":
		conn, err := ldap.DialURL(cfg.URL)
		if err != nil {
			return nil, err
		}
		if err := conn.StartTLS(tlsCfg); err != nil {
			conn.Close()
			return nil, err
		}
		return conn, nil
	default: // "none"
		return ldap.DialURL(cfg.URL)
	}
}
