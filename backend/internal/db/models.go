package db

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"

    "github.com/jmoiron/sqlx"
)

// -----------------------------
// Data models
// -----------------------------

// CASettings holds global CA configuration stored in DB.
type CASettings struct {
    ID              int64     `db:"id"`
    CAURL           string    `db:"ca_url"`
    RootFingerprint string    `db:"root_fingerprint"`
    ProvisionerName string    `db:"provisioner_name"`
    ACMEDirectories string    `db:"acme_directories"` // JSON array stored as text
    CreatedAt       time.Time `db:"created_at"`
    UpdatedAt       time.Time `db:"updated_at"`
}

// Provisioner represents a provisioner configured via UI.
// Note: provisioner secret is NOT persisted.
type Provisioner struct {
    Name           string    `db:"name"`
    Type           string    `db:"type"`
    ACMEDirectories string   `db:"acme_directories"` // JSON array as text
    CreatedAt      time.Time `db:"created_at"`
}

// Certificate represents a certificate record.
// We intentionally DO NOT store private_key_pem here.
type Certificate struct {
    ID            string         `db:"id"`
    CommonName    string         `db:"common_name"`
    DNSNames      string         `db:"dns_names"` // JSON array as text
    Serial        string         `db:"serial"`
    NotBefore     time.Time      `db:"not_before"`
    NotAfter      time.Time      `db:"not_after"`
    CertificatePEM sql.NullString `db:"certificate_pem"`
    CAChainPEM    sql.NullString `db:"ca_chain_pem"`
    Status        string         `db:"status"` // active | revoked
    CreatedAt     time.Time      `db:"created_at"`
    RevokedAt     sql.NullTime   `db:"revoked_at"`
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

// -----------------------------
// CRUD interface (implemented in db.go)
// -----------------------------

// The following functions are implemented in db.go and exposed here for consumers:
//
// func InitDB(ctx context.Context, dsn string) (*DB, error)
// func (db *DB) Close() error
//
// CA settings
// func (db *DB) GetCASettings(ctx context.Context) (*CASettings, error)
// func (db *DB) UpdateCASettings(ctx context.Context, s *CASettings) error
//
// Provisioners
// func (db *DB) ListProvisioners(ctx context.Context) ([]Provisioner, error)
// func (db *DB) GetProvisioner(ctx context.Context, name string) (*Provisioner, error)
// func (db *DB) CreateProvisioner(ctx context.Context, p *Provisioner) error
// func (db *DB) DeleteProvisioner(ctx context.Context, name string) error
//
// Certificates
// func (db *DB) ListCertificates(ctx context.Context) ([]Certificate, error)
// func (db *DB) GetCertificate(ctx context.Context, id string) (*Certificate, error)
// func (db *DB) CreateCertificate(ctx context.Context, c *Certificate) error
// func (db *DB) RevokeCertificate(ctx context.Context, serial string) error
// func (db *DB) DeleteCertificate(ctx context.Context, id string) error
//
// Audit
// func (db *DB) LogAudit(ctx context.Context, action, user, details, ip string) error
