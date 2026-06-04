package step

import (
    "archive/zip"
    "bytes"
    "crypto/x509"
    "encoding/json"
    "encoding/pem"
    "fmt"
    "io"
    "net/http"
    "path/filepath"
    "strings"
    "time"

    "github.com/DavidPik/step-ui/backend/internal/db"
)

type StepClient struct {
    CAURL             string
    RootFingerprint   string
    ProvisionerName   string
    ProvisionerSecret string
    ACMEDirectories   []string
    httpClient        *http.Client
}

// ------------------------------------------------------------
// Inicializace klienta
// ------------------------------------------------------------

func NewClientFromSettings(settings *db.CASettings) *StepClient {
    return &StepClient{
        CAURL:             settings.CAURL,
        RootFingerprint:   settings.RootFingerprint,
        ProvisionerName:   settings.ProvisionerName,
        ProvisionerSecret: settings.ProvisionerSecret,
        ACMEDirectories:   settings.ACMEDirectories,
        httpClient: &http.Client{
            Timeout: 20 * time.Second,
        },
    }
}

func (c *StepClient) UseProvisioner(name, secret string) {
    c.ProvisionerName = name
    c.ProvisionerSecret = secret
}

// ------------------------------------------------------------
// Provisionery
// ------------------------------------------------------------

type Provisioner struct {
    Name string `json:"name"`
    Type string `json:"type"`
}

type provisionerListResponse struct {
    Provisioners []Provisioner `json:"provisioners"`
}

func (c *StepClient) ListProvisioners() ([]Provisioner, error) {
    url := fmt.Sprintf("%s/provisioners", strings.TrimRight(c.CAURL, "/"))

    req, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("creating request for provisioners: %w", err)
    }
    req.Header.Set("Accept", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request to step-ca /provisioners failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("step-ca /provisioners returned %d: %s", resp.StatusCode, string(body))
    }

    var out provisionerListResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return nil, fmt.Errorf("decoding provisioner list: %w", err)
    }

    return out.Provisioners, nil
}

// ------------------------------------------------------------
// Vydání certifikátu
// ------------------------------------------------------------

type CertificateRequest struct {
    CommonName string   `json:"common_name"`
    DNSNames   []string `json:"dns_names"`
}

type CertificateResponse struct {
    Certificate string `json:"crt"` // PEM
    PrivateKey  string `json:"key"` // PEM
    CABundle    string `json:"ca"`  // PEM
}

// IssueCertificate zavolá step-ca a vrátí PEM cert, key, ca.
func (c *StepClient) IssueCertificate(req CertificateRequest) (*CertificateResponse, error) {
    url := fmt.Sprintf("%s/sign", strings.TrimRight(c.CAURL, "/"))

    payload := map[string]interface{}{
        "common_name": req.CommonName,
        "dns_names":   req.DNSNames,
        "provisioner": c.ProvisionerName,
        "password":    c.ProvisionerSecret,
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("marshal sign payload: %w", err)
    }

    httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("creating sign request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("request to step-ca /sign failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("step-ca /sign returned %d: %s", resp.StatusCode, string(b))
    }

    var out CertificateResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return nil, fmt.Errorf("decoding sign response: %w", err)
    }

    return &out, nil
}

// ------------------------------------------------------------
// Revokace certifikátu
// ------------------------------------------------------------

func (c *StepClient) RevokeCertificate(serial string) error {
    url := fmt.Sprintf("%s/revoke", strings.TrimRight(c.CAURL, "/"))

    payload := map[string]string{
        "serial":      serial,
        "provisioner": c.ProvisionerName,
        "password":    c.ProvisionerSecret,
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("marshal revoke payload: %w", err)
    }

    req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("creating revoke request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("request to step-ca /revoke failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("step-ca /revoke returned %d: %s", resp.StatusCode, string(b))
    }

    return nil
}

// ------------------------------------------------------------
// Balíček certifikátu (PEM/DER/CRT + key + CA + README)
// ------------------------------------------------------------

type PackageFormat string

const (
    PackageFormatZIP PackageFormat = "zip"
)

// BuildCertificatePackage vytvoří ZIP s různými formáty certifikátu.
func (c *StepClient) BuildCertificatePackage(commonName string, resp *CertificateResponse) ([]byte, error) {
    buf := &bytes.Buffer{}
    zipWriter := zip.NewWriter(buf)

    // Normalizace jména souboru
    base := sanitizeFilename(commonName)
    if base == "" {
        base = "certificate"
    }

    // PEM cert
    if err := addFileToZip(zipWriter, fmt.Sprintf("%s.pem", base), []byte(resp.Certificate)); err != nil {
        return nil, err
    }

    // CRT (jen jiná přípona PEM)
    if err := addFileToZip(zipWriter, fmt.Sprintf("%s.crt", base), []byte(resp.Certificate)); err != nil {
        return nil, err
    }

    // DER
    der, err := pemToDER([]byte(resp.Certificate))
    if err == nil && len(der) > 0 {
        if err := addFileToZip(zipWriter, fmt.Sprintf("%s.der", base), der); err != nil {
            return nil, err
        }
    }

    // Private key (PEM)
    if err := addFileToZip(zipWriter, fmt.Sprintf("%s.key", base), []byte(resp.PrivateKey)); err != nil {
        return nil, err
    }

    // CA bundle
    if err := addFileToZip(zipWriter, "ca.crt", []byte(resp.CABundle)); err != nil {
        return nil, err
    }

    // README
    readme := GenerateCertificateReadme(commonName)
    if err := addFileToZip(zipWriter, "README.txt", []byte(readme)); err != nil {
        return nil, err
    }

    if err := zipWriter.Close(); err != nil {
        return nil, fmt.Errorf("closing zip writer: %w", err)
    }

    return buf.Bytes(), nil
}

func addFileToZip(z *zip.Writer, name string, data []byte) error {
    f, err := z.Create(filepath.ToSlash(name))
    if err != nil {
        return fmt.Errorf("create zip entry %s: %w", name, err)
    }
    if _, err := f.Write(data); err != nil {
        return fmt.Errorf("write zip entry %s: %w", name, err)
    }
    return nil
}

func pemToDER(pemBytes []byte) ([]byte, error) {
    block, _ := pem.Decode(pemBytes)
    if block == nil {
        return nil, fmt.Errorf("failed to decode PEM")
    }
    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        return nil, fmt.Errorf("parse certificate: %w", err)
    }
    return cert.Raw, nil
}

// ------------------------------------------------------------
// Certificate metadata parser
// ------------------------------------------------------------

type CertificateMetadata struct {
    Serial    string
    NotBefore time.Time
    NotAfter  time.Time
}

func ParseCertificateMetadata(pemCert string) (*CertificateMetadata, error) {
    block, _ := pem.Decode([]byte(pemCert))
    if block == nil {
        return nil, fmt.Errorf("failed to decode PEM certificate")
    }

    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil {
        return nil, fmt.Errorf("failed to parse certificate: %w", err)
    }

    return &CertificateMetadata{
        Serial:    cert.SerialNumber.String(),
        NotBefore: cert.NotBefore,
        NotAfter:  cert.NotAfter,
    }, nil
}

func sanitizeFilename(s string) string {
    s = strings.TrimSpace(s)
    s = strings.ReplaceAll(s, " ", "_")
    s = strings.ReplaceAll(s, "/", "_")
    s = strings.ReplaceAll(s, "\\", "_")
    s = strings.ReplaceAll(s, ":", "_")
    s = strings.ReplaceAll(s, "*", "_")
    s = strings.ReplaceAll(s, "?", "_")
    s = strings.ReplaceAll(s, "\"", "_")
    s = strings.ReplaceAll(s, "<", "_")
    s = strings.ReplaceAll(s, ">", "_")
    s = strings.ReplaceAll(s, "|", "_")
    return s
}

// ------------------------------------------------------------
// README generator
// ------------------------------------------------------------

func GenerateCertificateReadme(commonName string) string {
    base := sanitizeFilename(commonName)
    if base == "" {
        base = "certificate"
    }

    return fmt.Sprintf(`# Certificate issued for %s

## Files

- %s.pem  – certificate in PEM format
- %s.crt  – certificate in PEM format (CRT extension)
- %s.der  – certificate in DER (binary) format
- %s.key  – private key in PEM format
- ca.crt  – CA bundle

## Usage examples

### NGINX

ssl_certificate     %s.crt;
ssl_certificate_key %s.key;
ssl_trusted_certificate ca.crt;

### Apache

SSLCertificateFile      %s.crt
SSLCertificateKeyFile   %s.key
SSLCACertificateFile    ca.crt

### Linux (system-wide)

sudo cp %s.crt /etc/ssl/certs/
sudo cp %s.key /etc/ssl/private/
sudo cp ca.crt /etc/ssl/certs/

`, commonName,
        base, base, base, base,
        base, base,
        base, base,
        base, base,
    )
}
