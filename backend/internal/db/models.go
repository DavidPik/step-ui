package db

import "time"

// -----------------------------
// Data models (kompatibilní s db.go)
// -----------------------------

// Provisioner reprezentuje provisioner nakonfigurovaný přes UI.
// Secret se neukládá do DB.
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

// Certificate reprezentuje záznam o certifikátu.
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

// AuditEvent loguje akce v systému.
type AuditEvent struct {
    ID        int64     `json:"id,omitempty"`
    Timestamp time.Time `json:"timestamp,omitempty"`
    Action    string    `json:"action,omitempty"`
    User      string    `json:"user,omitempty"`
    Details   string    `json:"details,omitempty"`
    IP        string    `json:"ip,omitempty"`
}
