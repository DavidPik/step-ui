package db

import (
    "time"
)

// ------------------------------------------------------------
// CA SETTINGS
// ------------------------------------------------------------

type CASettings struct {
    ID                uint      `gorm:"primaryKey" json:"id"`

    // Core CA configuration
    CAURL             string    `json:"ca_url"`
    RootFingerprint   string    `json:"root_fingerprint"`

    // Provisioner management
    ProvisionerName   string    `json:"provisioner_name"`
    ProvisionerSecret string    `json:"provisioner_secret"`

    // ACME directories (stored as JSON text)
    ACMEDirectories   []string  `gorm:"type:text" json:"acme_directories"`

    CreatedAt         time.Time `json:"created_at"`
    UpdatedAt         time.Time `json:"updated_at"`
}

// ------------------------------------------------------------
// CERTIFICATE MODEL
// ------------------------------------------------------------

type Certificate struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    CommonName  string    `json:"common_name"`
    DNSNames    string    `json:"dns_names"` // comma-separated list
    Serial      string    `json:"serial"`
    NotBefore   time.Time `json:"not_before"`
    NotAfter    time.Time `json:"not_after"`
    CreatedAt   time.Time `json:"created_at"`
}

// ------------------------------------------------------------
// AUDIT LOG
// ------------------------------------------------------------

type AuditEvent struct {
    ID        uint      `gorm:"primaryKey" json:"id"`

    // Who performed the action (optional)
    User      string    `json:"user"`

    // What happened (certificate_issued, certificate_revoked, settings_updated, provisioner_selected, etc.)
    Action    string    `json:"action"`

    // Optional details (e.g. CN, serial, old/new values)
    Details   string    `json:"details"`

    CreatedAt time.Time `json:"created_at"`
}
