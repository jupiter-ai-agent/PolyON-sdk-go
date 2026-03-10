// Package directory provides LDAP user/group lookup for PolyON modules.
//
// Uses the PRC directory credentials (Samba AD DC) from environment variables.
// This is a lightweight wrapper — for advanced LDAP operations, use go-ldap directly.
package directory

import (
	polyon "github.com/jupiter-ai-agent/PolyON-sdk-go"
)

// Config holds LDAP connection settings from PRC.
type Config struct {
	Host         string
	Port         int
	BaseDN       string
	BindDN       string
	BindPassword string
}

// NewConfig creates a directory Config from PRC config.
func NewConfig(cfg polyon.DirectoryConfig) *Config {
	return &Config{
		Host:         cfg.Host,
		Port:         cfg.Port,
		BaseDN:       cfg.BaseDN,
		BindDN:       cfg.BindDN,
		BindPassword: cfg.BindPassword,
	}
}

// User represents an AD user.
type User struct {
	DN                string
	SAMAccountName    string
	UserPrincipalName string
	DisplayName       string
	Mail              string
	MemberOf          []string
}

// Group represents an AD group.
type Group struct {
	DN             string
	SAMAccountName string
	Description    string
	Members        []string
}

// ConnectionURL returns the LDAP connection URL.
func (c *Config) ConnectionURL() string {
	if c.Port == 636 {
		return "ldaps://" + c.Host + ":636"
	}
	return "ldap://" + c.Host + ":389"
}

// Note: Actual LDAP operations require github.com/go-ldap/ldap/v3.
// This package provides config helpers and type definitions.
// For full LDAP client, see the integration guide:
// https://github.com/jupiter-ai-agent/PolyON-platform/blob/main/docs/integration-guide.md
