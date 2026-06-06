package db

import (
    "strings"
    "time"

    "gorm.io/gorm"
)

type Certificate struct {
    ID             uint      `gorm:"primaryKey" json:"id"`
    CommonName     string    `json:"common_name"`
    DNSNames       string    `json:"dns_names"`
    Serial         string    `json:"serial"`
    NotBefore      time.Time `json:"not_before"`
    NotAfter       time.Time `json:"not_after"`

    CertificatePEM string `gorm:"type:longtext" json:"certificate_pem"`
    PrivateKeyPEM  string `gorm:"type:longtext" json:"private_key_pem"`
    CAChainPEM     string `gorm:"type:longtext" json:"ca_chain_pem"`

    CreatedAt time.Time `json:"created_at"`
}

type AuditEvent struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    User      string    `json:"user"`
    Action    string    `json:"action"`
    Details   string    `json:"details"`
    CreatedAt time.Time `json:"created_at"`
}

type CASettings struct {
    ID                uint      `gorm:"primaryKey" json:"id"`
    CAURL             string    `json:"ca_url"`
    RootFingerprint   string    `json:"root_fingerprint"`
    ProvisionerName   string    `json:"provisioner_name"`
    ProvisionerSecret string    `json:"provisioner_secret"`
    // ACMEDirectories stored as CSV in DB column "acme_directories"
    ACMEDirectoriesCSV string   `gorm:"column:acme_directories" json:"-"`
    ACMEDirectories    []string `gorm:"-" json:"acme_directories"`

    CreatedAt time.Time `json:"created_at"`
}

func (c *CASettings) BeforeSave(tx *gorm.DB) error {
    c.ACMEDirectoriesCSV = strings.Join(c.ACMEDirectories, ",")
    return nil
}

func (c *CASettings) AfterFind(tx *gorm.DB) error {
    if c.ACMEDirectoriesCSV == "" {
        c.ACMEDirectories = []string{}
    } else {
        c.ACMEDirectories = strings.Split(c.ACMEDirectoriesCSV, ",")
    }
    return nil
}

// Provisioner stores metadata only. Secrets are NOT persisted.
type Provisioner struct {
    ID                 uint      `gorm:"primaryKey" json:"id"`
    Name               string    `gorm:"uniqueIndex" json:"name"`
    Type               string    `json:"type"`
    ACMEDirectoriesCSV string    `gorm:"column:acme_directories" json:"-"`
    ACMEDirectories    []string  `gorm:"-" json:"acme_directories"`
    CreatedAt          time.Time `json:"created_at"`
}

func (p *Provisioner) BeforeSave(tx *gorm.DB) error {
    p.ACMEDirectoriesCSV = strings.Join(p.ACMEDirectories, ",")
    return nil
}

func (p *Provisioner) AfterFind(tx *gorm.DB) error {
    if p.ACMEDirectoriesCSV == "" {
        p.ACMEDirectories = []string{}
    } else {
        p.ACMEDirectories = strings.Split(p.ACMEDirectoriesCSV, ",")
    }
    return nil
}
