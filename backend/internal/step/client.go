package step

import (
    "archive/zip"
    "bytes"
    "crypto/tls"
    "crypto/x509"
    "encoding/json"
    "encoding/pem"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/DavidPik/step-ui/backend/internal/db"
)

// StepClient komunikuje se step-ca.
// Poznámka: instance může být sdílena, proto chráníme mutable pole mutexem.
type StepClient struct {
    CAURL           string
    RootFingerprint string

    // provisioner fields are mutable and protected by mu
    mu                sync.RWMutex
    ProvisionerName   string
    ProvisionerSecret string

    ACMEDirectories []string
    httpClient      *http.Client
}

// NewClientFromSettings vytvoří klienta z nastavení DB.
// Timeout a TLS chování lze konfigurovat přes env proměnné:
// STEP_CLIENT_TIMEOUT (sekundy, default 20)
// STEP_CA_INSECURE (true/false, default false) - pokud true, TLS verify se vypne (pouze v izolovaných sítích)
func NewClientFromSettings(settings *db.CASettings) *StepClient {
    timeout := 20 * time.Second
    if t := strings.TrimSpace(os.Getenv("STEP_CLIENT_TIMEOUT")); t != "" {
        if sec, err := time.ParseDuration(t + "s"); err == nil {
            timeout = sec
        }
    }

    insecure := false
    if strings.ToLower(strings.TrimSpace(os.Getenv("STEP_CA_INSECURE"))) == "true" {
        insecure = true
    }

    // transport s možností vypnout TLS verify (explicitně přes env)
    tr := &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        TLSHandshakeTimeout: 10 * time.Second,
    }

    if insecure {
        tr.TLSClientConfig = &tls.Config{
            InsecureSkipVerify: true,
        }
    }

    client := &http.Client{
        Timeout:   timeout,
        Transport: tr,
    }

    c := &StepClient{
        CAURL:           settings.CAURL,
        RootFingerprint: settings.RootFingerprint,
        ACMEDirectories: settings.ACMEDirectories,
        httpClient:      client,
    }

    // initial provisioner from settings (if present)
    c.mu.Lock()
    c.ProvisionerName = settings.ProvisionerName
    c.ProvisionerSecret = "" // do not persist secrets from settings
    c.mu.Unlock()

    return c
}

// UseProvisioner nastaví provisioner a secret do paměti klienta.
// Pokud sdílíš klienta mezi gorutinami, toto je thread-safe.
func (c *StepClient) UseProvisioner(name, secret string) {
    c.mu.Lock()
    c.ProvisionerName = name
    c.ProvisionerSecret = secret
    c.mu.Unlock()
}

// getProvisionerCredentials vrátí aktuální provisioner name a secret (thread-safe).
func (c *StepClient) getProvisionerCredentials() (string, string) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.ProvisionerName, c.ProvisionerSecret
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

// ListProvisioners načte seznam provisionerů ze step-ca.
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

    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("step-ca /provisioners returned %d: %s", resp.StatusCode, string(body))
    }

    var out provisionerListResponse
    if err := json.Unmarshal(body, &out); err != nil {
        return nil, fmt.Errorf("decoding provisioner list: %w", err)
    }

    return out.Provisioners, nil
}

// GetProvisioner načte detail provisioneru best-effort (pokud step-ca nemá dedicated endpoint, hledá v listu).
func (c *StepClient) GetProvisioner(name string) (*Provisioner, error) {
    provs, err := c.ListProvisioners()
    if err != nil {
        return nil, err
    }
    for _, p := range provs {
        if p.Name == name {
            return &p, nil
        }
    }
    return nil, fmt.Errorf("provisioner %s not found in step-ca", name)
}

// CreateProvisioner vytvoří provisioner v step-ca.
// payload může obsahovat "secret" nebo "password" podle potřeby.
// Secret je předán pouze v tomto volání a není nikde persistován.
func (c *StepClient) CreateProvisioner(payload map[string]interface{}) error {
    url := fmt.Sprintf("%s/provisioners", strings.TrimRight(c.CAURL, "/"))

    body, _ := json.Marshal(payload)
    req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("creating request to step-ca /provisioners: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("request to step-ca /provisioners failed: %w", err)
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("step-ca /provisioners returned %d: %s", resp.StatusCode, string(respBody))
    }

    return nil
}

// DeleteProvisioner smaže provisioner v step-ca.
// payload může obsahovat secret pokud step-ca vyžaduje autentizaci při smazání.
func (c *StepClient) DeleteProvisioner(name string, payload map[string]interface{}) error {
    url := fmt.Sprintf("%s/provisioners/%s", strings.TrimRight(c.CAURL, "/"), name)

    var req *http.Request
    var err error
    if payload != nil {
        body, _ := json.Marshal(payload)
        req, err = http.NewRequest(http.MethodDelete, url, bytes.NewReader(body))
        if err == nil {
            req.Header.Set("Content-Type", "application/json")
        }
    } else {
        req, err = http.NewRequest(http.MethodDelete, url, nil)
    }
    if err != nil {
        return fmt.Errorf("creating delete request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("request to step-ca DELETE /provisioners/%s failed: %w", name, err)
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("step-ca DELETE /provisioners/%s returned %d: %s", name, resp.StatusCode, string(respBody))
    }

    return nil
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
// Použije aktuální provisioner a secret z klienta (v paměti).
func (c *StepClient) IssueCertificate(req CertificateRequest) (*CertificateResponse, error) {
    url := fmt.Sprintf("%s/sign", strings.TrimRight(c.CAURL, "/"))

    provName, provSecret := c.getProvisionerCredentials()

    payload := map[string]interface{}{
        "common_name": req.CommonName,
        "dns_names":   req.DNSNames,
    }
    if provName != "" {
        payload["provisioner"] = provName
    }
    if provSecret != "" {
        payload["password"] = provSecret
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

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("step-ca /sign returned %d: %s", resp.StatusCode, string(respBody))
    }

    var out CertificateResponse
    if err := json.Unmarshal(respBody, &out); err != nil {
        return nil, fmt.Errorf("decoding sign response: %w", err)
    }

    return &out, nil
}

// ------------------------------------------------------------
// Revokace certifikátu
// ------------------------------------------------------------

func (c *StepClient) RevokeCertificate(serial string) error {
    url := fmt.Sprintf("%s/revoke", strings.TrimRight(c.CAURL, "/"))

    provName, provSecret := c.getProvisionerCredentials()

    payload := map[string]string{
        "serial": serial,
    }
    if provName != "" {
        payload["provisioner"] = provName
    }
    if provSecret != "" {
        payload["password"] = provSecret
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

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("step-ca /revoke returned %d: %s", resp.StatusCode, string(respBody))
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

func (c *StepClient) BuildCertificatePackage(commonName string, resp *CertificateResponse) ([]byte, error) {
    buf := &bytes.Buffer{}
    zipWriter := zip.NewWriter(buf)

    base := sanitizeFilename(commonName)
    if base == "" {
        base = "certificate"
    }

    if err := addFileToZip(zipWriter, fmt.Sprintf("%s.pem", base), []byte(resp.Certificate)); err != nil {
        return nil, err
    }
    if err := addFileToZip(zipWriter, fmt.Sprintf("%s.crt", base), []byte(resp.Certificate)); err != nil {
        return nil, err
    }

    der, err := pemToDER([]byte(resp.Certificate))
    if err == nil && len(der) > 0 {
        if err := addFileToZip(zipWriter, fmt.Sprintf("%s.der", base), der); err != nil {
            return nil, err
        }
    }

    if err := addFileToZip(zipWriter, fmt.Sprintf("%s.key", base), []byte(resp.PrivateKey)); err != nil {
        return nil, err
    }

    if err := addFileToZip(zipWriter, "ca.crt", []byte(resp.CABundle)); err != nil {
        return nil, err
    }

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

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

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

func getEnv(key, def string) string {
    if v := strings.TrimSpace(os.Getenv(key)); v != "" {
        return v
    }
    return def
}

// GenerateCertificateReadme returns a short README for packaged certificates.
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
