package db

import (
    "database/sql"
    "encoding/json"
    "time"
)

// -----------------------------
// Data models
// -----------------------------

// Provisioner represents a provisioner configured via UI.
// Note: provisioner secret is NOT persisted.
type Provisioner struct {
    Name             string    `db:"name"`
    Type             string    `db:"type"`
    JWK              string    `db:"jwk"`
    ACMEDirectories  []string  `db:"acme_directories"` // JSON array stored as text
    CreatedAt        time.Time `db:"created_at"`
}

// Certificate represents a certificate record.
type Certificate struct {
    ID             string         `db:"id"`
    CommonName     string         `db:"common_name"`
    DNSNames       string         `db:"dns_names"` // JSON array as text
    Serial         string         `db:"serial"`
    NotBefore      time.Time      `db:"not_before"`
    NotAfter       time.Time      `db:"not_after"`
    CertificatePEM sql.NullString `db:"certificate_pem"`
    CAChainPEM     sql.NullString `db:"ca_chain_pem"`
    CreatedAt      time.Time      `db:"created_at"`
}

// AuditEvent logs actions performed in the system.
type AuditEvent struct {
    ID        int64     `db:"id"`
    Timestamp time.Time `db:"timestamp"`
    Action    string    `db:"action"`
    User      string    `db:"user"`
    Details   string    `db:"details"`
    IP        string    `db:"ip"`
}

// -----------------------------
// Helper types for JSON fields
// -----------------------------

// StringArrayToJSON converts string slice to JSON string for storage.
func StringArrayToJSON(arr []string) (string, error) {
    if arr == nil {
        return "[]", nil
    }
    b, err := json.Marshal(arr)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

// JSONToStringArray converts stored JSON string to []string.
func JSONToStringArray(s string) ([]string, error) {
    if s == "" {
        return []string{}, nil
    }
    var out []string
    if err := json.Unmarshal([]byte(s), &out); err != nil {
        return nil, err
    }
    return out, nil
}
