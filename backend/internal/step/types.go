package step

// APIError represents an error returned by step-ca server API.
type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

// Provisioner represents a provisioner object returned by step-ca.
type Provisioner struct {
    Name           string   `json:"name"`
    Type           string   `json:"type"`
    JWK            string   `json:"jwk"`
    ACMEDirectories []string `json:"acme_directories"`
}

// IssueCertificateRequest is the minimal request shape used by the backend.
type IssueCertificateRequest struct {
    CommonName   string   `json:"common_name"`
    DNSNames     []string `json:"dns_names"`
    NotAfterDays int      `json:"not_after_days"`
    CSRPEM       string   `json:"csr"` // optional: if provided, CA will sign CSR
}

// IssueCertificateResponse is the minimal response shape we expect from step-ca /sign.
type IssueCertificateResponse struct {
    ID             string `json:"id"`
    CommonName     string `json:"common_name"`
    Serial         string `json:"serial"`
    CertificatePEM string `json:"crt"`
    KeyPEM         string `json:"key"`
    CABundlePEM    string `json:"chain"`
    NotBefore      string `json:"not_before"`
    NotAfter       string `json:"not_after"`
}

// SignCSRRequest / SignCSRResponse (kept for completeness)
type SignCSRRequest struct {
    CSRPEM      string `json:"csr"`
    NotAfterDays int   `json:"not_after_days"`
}
type SignCSRResponse struct {
    CertificatePEM string `json:"crt"`
    Serial         string `json:"serial"`
    NotBefore      string `json:"not_before"`
    NotAfter       string `json:"not_after"`
}

// RevokeResponse minimal shape
type RevokeResponse struct {
    Serial string `json:"serial"`
    Status string `json:"status"`
}
