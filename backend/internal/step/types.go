package step

// APIError represents an error returned by step-ca server API.
type APIError struct {
    Code    int    `json:"code,omitempty"`
    Message string `json:"message,omitempty"`
}

func (e *APIError) Error() string {
    return e.Message
}

// Provisioner represents a CA provisioner.
type Provisioner struct {
    Name            string   `json:"name"`
    Type            string   `json:"type"`
    ACMEDirectories []string `json:"acme_directories,omitempty"`
}

// IssueCertificateRequest represents a request to issue a certificate.
type IssueCertificateRequest struct {
    CommonName    string   `json:"common_name,omitempty"`
    DNSNames      []string `json:"dns_names,omitempty"`
    NotAfterDays  int      `json:"not_after_days,omitempty"`
}

// IssueCertificateResponse represents certificate metadata returned by step-ca.
type IssueCertificateResponse struct {
    ID            string `json:"id"`
    Serial        string `json:"serial"`
    CommonName    string `json:"common_name"`
    CertificatePEM string `json:"crt"`
    CABundlePEM    string `json:"ca"`
    NotBefore      string `json:"not_before"`
    NotAfter       string `json:"not_after"`
}

// SignCSRRequest represents a CSR signing request.
type SignCSRRequest struct {
    CSRPEM       string `json:"csr"`
    NotAfterDays int    `json:"not_after_days,omitempty"`
}

// SignCSRResponse represents a signed certificate returned by step-ca.
type SignCSRResponse struct {
    CertificatePEM string `json:"crt"`
    CABundlePEM    string `json:"ca"`
}

// RevokeResponse represents a revocation response.
type RevokeResponse struct {
    Status string `json:"status"`
}
