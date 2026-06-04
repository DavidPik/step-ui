package db

import "time"

type Certificate struct {
    ID             uint      `gorm:"primaryKey" json:"id"`
    CommonName     string    `json:"common_name"`
    DNSNames       string    `json:"dns_names"`
    Serial         string    `json:"serial"`
    NotBefore      time.Time `json:"not_before"`
    NotAfter       time.Time `json:"not_after"`

    // PEM data for download
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
    ID               uint     `gorm:"primaryKey" json:"id"`
    CAURL            string   `json:"ca_url"`
    RootFingerprint  string   `json:"root_fingerprint"`
    ProvisionerName  string   `json:"provisioner_name"`
    ProvisionerSecret string  `json:"provisioner_secret"`
    ACMEDirectories  []string `gorm:"-" json:"acme_directories"`

    CreatedAt time.Time `json:"created_at"`
}
