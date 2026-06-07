package step

import (
    "bytes"
    "context"
    "crypto/tls"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "strings"
    "sync"
    "time"
)

// -----------------------------
// Canonical server API paths (Smallstep step-ca server API)
// Centralized here so change is in one place.
// -----------------------------
const (
    pathProvisioners        = "/provisioners"
    pathProvisioner         = "/provisioners/%s"
    pathSelectProvisioner   = "/provisioners/%s/select"
    pathSign                = "/sign"
    pathRevoke              = "/revoke"
    pathCertificate         = "/certificates/%s"
    pathCertificateDownload = "/certificates/%s/download"
    pathVersion             = "/version"
    pathHealth              = "/health"
)

// -----------------------------
// API types used by client and main
// -----------------------------

// APIError represents an error returned by step-ca in JSON form.
func (e *APIError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    return fmt.Sprintf("step api error: code=%d", e.Code)
}

// -----------------------------
// Step client implementation
// -----------------------------

// StepClient is a simple HTTP client for communicating with step-ca server API.
// It is safe for concurrent use.
type StepClient struct {
    baseURL    *url.URL
    httpClient *http.Client
    mu         sync.Mutex // protects any future mutable fields
}

// Config holds configuration for StepClient.
type Config struct {
    BaseURL            string
    Timeout            time.Duration
    InsecureSkipVerify bool
    Headers            map[string]string

    // Optional HTTP client provided by caller (main.go). If non-nil, it will be used
    // for all requests. If nil, NewClient will create a default http.Client using
    // Timeout and InsecureSkipVerify.
    HTTPClient *http.Client
}

// NewClient creates a new StepClient configured with the provided Config.
// BaseURL must be a valid absolute URL (e.g., https://step-ca:9000).
// Timeout defaults to 10s if zero.
func NewClient(cfg Config) (*StepClient, error) {
    if strings.TrimSpace(cfg.BaseURL) == "" {
        return nil, errors.New("step: BaseURL is required")
    }
    parsed, err := url.Parse(cfg.BaseURL)
    if err != nil {
        return nil, fmt.Errorf("step: invalid base url: %w", err)
    }
    timeout := cfg.Timeout
    if timeout <= 0 {
        timeout = 10 * time.Second
    }

    // If caller provided an HTTP client, use it as-is (caller is responsible for TLS config and timeout).
    if cfg.HTTPClient != nil {
        // Ensure the provided client has a timeout; if not, set it to the configured timeout.
        if cfg.HTTPClient.Timeout == 0 {
            cfg.HTTPClient.Timeout = timeout
        }
        return &StepClient{
            baseURL:    parsed,
            httpClient: cfg.HTTPClient,
        }, nil
    }

    // Otherwise create a default transport and client (backwards-compatible behavior).
    tr := &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout:   5 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        ForceAttemptHTTP2:     true,
        MaxIdleConns:          100,
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   5 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    }

    if parsed.Scheme == "https" && cfg.InsecureSkipVerify {
        tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    }

    client := &http.Client{
        Transport: tr,
        Timeout:   timeout,
    }

    return &StepClient{
        baseURL:    parsed,
        httpClient: client,
    }, nil
}

// buildURL joins base URL with path and optional query values.
func (c *StepClient) buildURL(p string, q url.Values) string {
    u := *c.baseURL
    u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(p, "/")
    u.RawQuery = q.Encode()
    return u.String()
}

// doRequest performs an HTTP request with optional JSON body and decodes JSON response into out (if non-nil).
func (c *StepClient) doRequest(ctx context.Context, method, path string, query url.Values, body interface{}, out interface{}, headers map[string]string) error {
    var bodyReader io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil {
            return fmt.Errorf("step: marshal body: %w", err)
        }
        bodyReader = bytes.NewReader(b)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, query), bodyReader)
    if err != nil {
        return fmt.Errorf("step: new request: %w", err)
    }
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    req.Header.Set("Accept", "application/json")

    // merge headers: client-level (Config.Headers) are not stored currently; merge per-request headers
    for k, v := range headers {
        req.Header.Set(k, v)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("step: request failed: %w", err)
    }
    defer resp.Body.Close()

    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("step: read response: %w", err)
    }

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        var apiErr APIError
        _ = json.Unmarshal(respBytes, &apiErr)
        if apiErr.Message == "" {
            apiErr.Message = fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(respBytes))
        }
        apiErr.Code = resp.StatusCode
        return &apiErr
    }

    if out == nil || len(respBytes) == 0 {
        return nil
    }

    if err := json.Unmarshal(respBytes, out); err != nil {
        return fmt.Errorf("step: decode response: %w", err)
    }
    return nil
}

// ValidateServer performs lightweight checks against the step-ca server API.
// It calls /version and /provisioners to verify the server is reachable and responds.
func (c *StepClient) ValidateServer(ctx context.Context) error {
    // Check version
    var ver map[string]interface{}
    if err := c.doRequest(ctx, http.MethodGet, pathVersion, nil, nil, &ver, nil); err != nil {
        // try health as fallback
        var health map[string]interface{}
        if err2 := c.doRequest(ctx, http.MethodGet, pathHealth, nil, nil, &health, nil); err2 != nil {
            return fmt.Errorf("step: version/health check failed: version err: %v; health err: %v", err, err2)
        }
    }

    // Check provisioners endpoint (may require auth; return error so caller can log)
    var provResp struct {
        Items []Provisioner `json:"items"`
    }
    if err := c.doRequest(ctx, http.MethodGet, pathProvisioners, nil, nil, &provResp, nil); err != nil {
        return fmt.Errorf("step: provisioners endpoint check failed: %w", err)
    }
    return nil
}

// -----------------------------
// Provisioner methods (server API)
// -----------------------------

func (c *StepClient) ListProvisioners(ctx context.Context) ([]Provisioner, error) {
    var out struct {
        Items []Provisioner `json:"items"`
    }
    if err := c.doRequest(ctx, http.MethodGet, pathProvisioners, nil, nil, &out, nil); err != nil {
        // fallback: try direct array decode
        var arr []Provisioner
        if err2 := c.doRequest(ctx, http.MethodGet, pathProvisioners, nil, nil, &arr, nil); err2 == nil {
            return arr, nil
        }
        return nil, err
    }
    return out.Items, nil
}

func (c *StepClient) GetProvisioner(ctx context.Context, name string) (*Provisioner, error) {
    if name == "" {
        return nil, errors.New("name required")
    }
    var p Provisioner
    path := fmt.Sprintf(pathProvisioner, url.PathEscape(name))
    if err := c.doRequest(ctx, http.MethodGet, path, nil, nil, &p, nil); err != nil {
        if apiErr, ok := err.(*APIError); ok && apiErr.Code == http.StatusNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &p, nil
}

func (c *StepClient) CreateProvisioner(ctx context.Context, p *Provisioner, secret string) error {
    if p == nil {
        return errors.New("provisioner required")
    }
    payload := map[string]interface{}{
        "name": p.Name,
        "type": p.Type,
    }
    if len(p.ACMEDirectories) > 0 {
        payload["acme_directories"] = p.ACMEDirectories
    }
    if secret != "" {
        payload["secret"] = secret
    }
    return c.doRequest(ctx, http.MethodPost, pathProvisioners, nil, payload, nil, nil)
}

func (c *StepClient) DeleteProvisioner(ctx context.Context, name string, secret string) error {
    if name == "" {
        return errors.New("name required")
    }
    payload := map[string]interface{}{}
    if secret != "" {
        payload["secret"] = secret
    }
    path := fmt.Sprintf(pathProvisioner, url.PathEscape(name))
    return c.doRequest(ctx, http.MethodDelete, path, nil, payload, nil, nil)
}

func (c *StepClient) SelectProvisioner(ctx context.Context, name string, secret string) error {
    if name == "" {
        return errors.New("name required")
    }
    path := fmt.Sprintf(pathSelectProvisioner, url.PathEscape(name))
    payload := map[string]interface{}{}
    if secret != "" {
        payload["secret"] = secret
    }
    return c.doRequest(ctx, http.MethodPost, path, nil, payload, nil, nil)
}

// -----------------------------
// Certificate operations (server API)
// -----------------------------

// IssueCertificate requests a new certificate. Uses canonical /sign endpoint.
func (c *StepClient) IssueCertificate(ctx context.Context, req IssueCertificateRequest, secret string) (*IssueCertificateResponse, error) {
    payload := map[string]interface{}{}
    if req.NotAfterDays > 0 {
        payload["not_after_days"] = req.NotAfterDays
    }
    if req.CommonName != "" || len(req.DNSNames) > 0 {
        subject := map[string]interface{}{}
        if req.CommonName != "" {
            subject["common_name"] = req.CommonName
        }
        if len(req.DNSNames) > 0 {
            subject["dns_names"] = req.DNSNames
        }
        payload["subject"] = subject
    }
    if req.CSRPEM != "" {
        payload["csr"] = req.CSRPEM
    }
    if secret != "" {
        payload["secret"] = secret
    }

    var out IssueCertificateResponse
    if err := c.doRequest(ctx, http.MethodPost, pathSign, nil, payload, &out, nil); err != nil {
        return nil, err
    }
    return &out, nil
}

// SignCSR sends a CSR to /sign for signing.
func (c *StepClient) SignCSR(ctx context.Context, req SignCSRRequest, secret string) (*SignCSRResponse, error) {
    payload := map[string]interface{}{
        "csr": req.CSRPEM,
    }
    if req.NotAfterDays > 0 {
        payload["not_after_days"] = req.NotAfterDays
    }
    if secret != "" {
        payload["secret"] = secret
    }
    var out SignCSRResponse
    if err := c.doRequest(ctx, http.MethodPost, pathSign, nil, payload, &out, nil); err != nil {
        return nil, err
    }
    return &out, nil
}

// GetCertificate fetches certificate metadata and PEMs by id.
func (c *StepClient) GetCertificate(ctx context.Context, id string) (*IssueCertificateResponse, error) {
    if id == "" {
        return nil, errors.New("id required")
    }
    var out IssueCertificateResponse
    path := fmt.Sprintf(pathCertificate, url.PathEscape(id))
    if err := c.doRequest(ctx, http.MethodGet, path, nil, nil, &out, nil); err != nil {
        if apiErr, ok := err.(*APIError); ok && apiErr.Code == http.StatusNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &out, nil
}

// DownloadCertificatePackage returns an absolute URL to download a ZIP package for the certificate.
func (c *StepClient) DownloadCertificatePackage(ctx context.Context, id string) (string, error) {
    if id == "" {
        return "", errors.New("id required")
    }
    path := fmt.Sprintf(pathCertificateDownload, url.PathEscape(id))
    return c.buildURL(path, nil), nil
}

// RevokeCertificate revokes a certificate by serial using canonical /revoke endpoint.
func (c *StepClient) RevokeCertificate(ctx context.Context, serial string, secret string) (*RevokeResponse, error) {
    if serial == "" {
        return nil, errors.New("serial required")
    }
    payload := map[string]interface{}{
        "serial": serial,
    }
    if secret != "" {
        payload["secret"] = secret
    }
    var out RevokeResponse
    if err := c.doRequest(ctx, http.MethodPost, pathRevoke, nil, payload, &out, nil); err != nil {
        return nil, err
    }
    return &out, nil
}
