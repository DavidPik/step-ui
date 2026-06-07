package db

import (
    "encoding/json"
    "time"
)

// -----------------------------
// Data models (compatible with db.go usage)
// -----------------------------

// Provisioner represents a provisioner configured via UI.
// Note: provisioner secret is NOT persisted.
type Provisioner struct {
    Name            string    `json:"name"`
    Type            string    `json:"type"`
    JWK             string    `json:"jwk,omitempty"`
    ACMEDirectories []string  `json:"acme_directories,omitempty"`
    CreatedAt       time.Time `json:"created_at,omitempty"`
}

type ProvisionerStatus struct {
    Name   string `json:"name"`
    Status string `json:"status"`
}

type CASettings struct {
    ProvisionerName string    `json:"provisioner_name,omitempty"`
    ACMEDirectories []string  `json:"acme_directories,omitempty"`
    UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// Certificate represents a certificate record.
type Certificate struct {
    ID             string    `json:"id"`
    CommonName     string    `json:"common_name"`
    DNSNames       []string  `json:"dns_names,omitempty"`
    Serial         string    `json:"serial,omitempty"`
    NotBefore      time.Time `json:"not_before,omitempty"`
    NotAfter       time.Time `json:"not_after,omitempty"`
    CertificatePEM string    `json:"certificate_pem,omitempty"`
    PrivateKeyPEM  string    `json:"private_key_pem,omitempty"`
    CAChainPEM     string    `json:"ca_chain_pem,omitempty"`
    Status         string    `json:"status,omitempty"`
    CreatedAt      time.Time `json:"created_at,omitempty"`
}

// AuditEvent logs actions performed in the system.
type AuditEvent struct {
    ID        int64     `json:"id,omitempty"`
    Timestamp time.Time `json:"timestamp,omitempty"`
    Action    string    `json:"action,omitempty"`
    User      string    `json:"user,omitempty"`
    Details   string    `json:"details,omitempty"`
    IP        string    `json:"ip,omitempty"`
}

// -----------------------------
// Helper functions for JSON fields
// -----------------------------
// Note: db.go expects helper functions named marshalStringArray / unmarshalStringArray.

func marshalStringArray(a []string) (string, error) {
    if a == nil {
        return "[]", nil
    }
    b, err := json.Marshal(a)
    return string(b), err
}

func unmarshalStringArray(s string) ([]string, error) {
    if s == "" {
        return []string{}, nil
    }
    var out []string
    if err := json.Unmarshal([]byte(s), &out); err != nil {
        return nil, err
    }
    return out, nil
}
