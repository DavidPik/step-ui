package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/gorilla/mux"

    "github.com/DavidPik/step-ui/backend/internal/db"
    "github.com/DavidPik/step-ui/backend/internal/step"
)

// NOTE: adjust module import paths above to match your go.mod module path.
// The manifest expects backend/internal/db and backend/step packages.

const (
    envDBDSN           = "DB_DSN"
    envDatabaseDSN     = "DATABASE_DSN"
    envStepURL         = "STEP_CA_URL"
    envStepInsecure    = "STEP_INSECURE_SKIP_VERIFY"
    envStepTimeout     = "STEP_TIMEOUT_SECONDS"
    envListenAddr      = "LISTEN_ADDR"
    defaultListen      = "0.0.0.0:8080"
    defaultStepTimeout = 10
)

func main() {
    // Allow overriding via flags for local dev convenience
    var (
        flagListen = flag.String("listen", "", "listen address (overrides LISTEN_ADDR env)")
    )
    flag.Parse()

    // Basic logger to stdout (visible in container logs / Portainer)
    logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

    // Read configuration from env (support both DB_DSN and DATABASE_DSN)
    dsn := strings.TrimSpace(os.Getenv(envDBDSN))
    if dsn == "" {
        dsn = strings.TrimSpace(os.Getenv(envDatabaseDSN))
    }
    stepURL := strings.TrimSpace(os.Getenv(envStepURL))
    insecure := strings.TrimSpace(os.Getenv(envStepInsecure))
    stepTimeoutSec := defaultStepTimeout
    if v := strings.TrimSpace(os.Getenv(envStepTimeout)); v != "" {
        if t, err := parseInt(v); err == nil && t > 0 {
            stepTimeoutSec = t
        }
    }
    listen := defaultListen
    if v := strings.TrimSpace(os.Getenv(envListenAddr)); v != "" {
        listen = v
    }
    if *flagListen != "" {
        listen = *flagListen
    }

    // Validate required envs
    if dsn == "" {
        logger.Fatal("DB_DSN or DATABASE_DSN is required")
    }
    if stepURL == "" {
        logger.Fatal("STEP_CA_URL is required")
    }

    // Initialize Step client
    stepCfg := step.Config{
        BaseURL:            stepURL,
        Timeout:            time.Duration(stepTimeoutSec) * time.Second,
        InsecureSkipVerify: strings.EqualFold(insecure, "true") || insecure == "1",
    }
    stepClient, err := step.NewClient(stepCfg)
    if err != nil {
        logger.Fatalf("failed to create step client: %v", err)
    }

    // Validate step server (lightweight checks). Log errors but continue startup.
    ctxValidate, cancelValidate := context.WithTimeout(context.Background(), 8*time.Second)
    defer cancelValidate()
    if err := stepClient.ValidateServer(ctxValidate); err != nil {
        logger.Printf("WARN: step server validation failed: %v", err)
    } else {
        logger.Printf("step server validation succeeded")
    }

    // Initialize DB
    ctxDB, cancelDB := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancelDB()
    database, err := db.InitDB(ctxDB, dsn)
    if err != nil {
        logger.Fatalf("failed to initialize database: %v", err)
    }
    // Ensure DB closed on exit
    defer func() {
        if err := database.Close(); err != nil {
            logger.Printf("error closing db: %v", err)
        }
    }()

    // Router and handlers
    r := mux.NewRouter()

    // Health
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    }).Methods(http.MethodGet)

    // API prefix
    api := r.PathPrefix("/api").Subrouter()

    // Provisioners: list
    api.HandleFunc("/provisioners", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        provs, err := database.ListProvisioners(ctx)
        if err != nil {
            logger.Printf("error listing provisioners: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to list provisioners")
            return
        }
        // frontend expects { items: [...] }
        writeJSON(w, map[string]interface{}{"items": provs})
    }).Methods(http.MethodGet)

    // Provisioner create
    api.HandleFunc("/provisioners", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        var p db.Provisioner
        if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
            httpError(w, http.StatusBadRequest, "invalid request body")
            return
        }
        if p.Name == "" || p.Type == "" {
            httpError(w, http.StatusBadRequest, "name and type are required")
            return
        }
        if err := database.CreateProvisioner(ctx, &p); err != nil {
            logger.Printf("error creating provisioner: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to create provisioner")
            return
        }
        _ = database.LogAudit(ctx, "provisioner.create", "system", fmt.Sprintf("created provisioner %s", p.Name), r.RemoteAddr)
        w.WriteHeader(http.StatusCreated)
        writeJSON(w, map[string]string{"result": "ok"})
    }).Methods(http.MethodPost)

    // Provisioner delete (expects JSON body with optional secret)
    api.HandleFunc("/provisioners/{name}", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        vars := mux.Vars(r)
        name := vars["name"]
        if name == "" {
            httpError(w, http.StatusBadRequest, "name required")
            return
        }
        // Read optional secret from body (but we do not persist it)
        var body map[string]interface{}
        _ = json.NewDecoder(r.Body).Decode(&body)
        secret := ""
        if s, ok := body["secret"].(string); ok {
            secret = s
        }

        // First attempt to delete in step-ca (best-effort). If step returns error, log and return error.
        if err := stepClient.DeleteProvisioner(ctx, name, secret); err != nil {
            // If step returns not found, still attempt DB delete to keep UI consistent.
            if apiErr, ok := err.(*step.APIError); ok && apiErr.Code == http.StatusNotFound {
                logger.Printf("provisioner %s not found in step-ca; proceeding to DB delete", name)
            } else {
                logger.Printf("error deleting provisioner in step-ca: %v", err)
                httpError(w, http.StatusBadGateway, "failed to delete provisioner in CA")
                return
            }
        }

        if err := database.DeleteProvisioner(ctx, name); err != nil {
            logger.Printf("error deleting provisioner in DB: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to delete provisioner")
            return
        }
        _ = database.LogAudit(ctx, "provisioner.delete", "system", fmt.Sprintf("deleted provisioner %s", name), r.RemoteAddr)
        writeJSON(w, map[string]string{"result": "ok"})
    }).Methods(http.MethodDelete)

    // Select provisioner (activate) - expects optional secret in body
    api.HandleFunc("/provisioners/{name}/select", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        vars := mux.Vars(r)
        name := vars["name"]
        if name == "" {
            httpError(w, http.StatusBadRequest, "name required")
            return
        }
        var body map[string]interface{}
        _ = json.NewDecoder(r.Body).Decode(&body)
        secret := ""
        if s, ok := body["secret"].(string); ok {
            secret = s
        }

        if err := stepClient.SelectProvisioner(ctx, name, secret); err != nil {
            logger.Printf("error selecting provisioner in step-ca: %v", err)
            httpError(w, http.StatusBadGateway, "failed to select provisioner in CA")
            return
        }

        // Update CA settings row to reflect selected provisioner
        settings, err := database.GetCASettings(ctx)
        if err != nil {
            logger.Printf("error reading ca settings: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to update settings")
            return
        }
        if settings == nil {
            httpError(w, http.StatusInternalServerError, "ca settings missing")
            return
        }
        settings.ProvisionerName = name
        if err := database.UpdateCASettings(ctx, settings); err != nil {
            logger.Printf("error updating ca settings: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to update settings")
            return
        }
        _ = database.LogAudit(ctx, "provisioner.select", "system", fmt.Sprintf("selected provisioner %s", name), r.RemoteAddr)
        writeJSON(w, map[string]string{"result": "ok"})
    }).Methods(http.MethodPost)

    // Certificates: list
    api.HandleFunc("/certificates", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        certs, err := database.ListCertificates(ctx)
        if err != nil {
            logger.Printf("error listing certificates: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to list certificates")
            return
        }
        // frontend expects { items: [...] }
        writeJSON(w, map[string]interface{}{"items": certs})
    }).Methods(http.MethodGet)

    // Certificate detail
    api.HandleFunc("/certificates/{id}", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        id := mux.Vars(r)["id"]
        if id == "" {
            httpError(w, http.StatusBadRequest, "id required")
            return
        }
        cert, err := database.GetCertificate(ctx, id)
        if err != nil {
            logger.Printf("error getting certificate: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to get certificate")
            return
        }
        if cert == nil {
            httpError(w, http.StatusNotFound, "certificate not found")
            return
        }
        writeJSON(w, cert)
    }).Methods(http.MethodGet)

    // Issue certificate (proxy to step /sign). Expects JSON payload with common_name, dns_names, optional secret.
    api.HandleFunc("/certificates", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        var payload struct {
            CommonName   string   `json:"common_name"`
            DNSNames     []string `json:"dns_names"`
            Secret       string   `json:"secret"`
            NotAfterDays int      `json:"not_after_days"`
        }
        if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
            httpError(w, http.StatusBadRequest, "invalid request body")
            return
        }
        req := step.IssueCertificateRequest{
            CommonName:   payload.CommonName,
            DNSNames:     payload.DNSNames,
            NotAfterDays: payload.NotAfterDays,
        }
        resp, err := stepClient.IssueCertificate(ctx, req, payload.Secret)
        if err != nil {
            logger.Printf("error issuing certificate: %v", err)
            httpError(w, http.StatusBadGateway, "failed to issue certificate")
            return
        }

        // Persist certificate metadata (do not store private key)
        c := &db.Certificate{
            ID:             resp.ID,
            CommonName:     resp.CommonName,
            DNSNames:       payload.DNSNames,
            Serial:         resp.Serial,
            Status:         "active",
            CertificatePEM: resp.CertificatePEM,
            CAChainPEM:     resp.CABundlePEM,
            NotBefore:      parseTimeOrNow(resp.NotBefore),
            NotAfter:       parseTimeOrNow(resp.NotAfter),
        }

        // CreateCertificate returns (id, error)
        if _, err := database.CreateCertificate(ctx, c); err != nil {
            logger.Printf("warning: failed to persist certificate metadata: %v", err)
            // continue; we still return the issued material to the caller
        }
        _ = database.LogAudit(ctx, "certificate.issue", "system", fmt.Sprintf("issued certificate %s serial=%s", resp.CommonName, resp.Serial), r.RemoteAddr)

        // Return the raw response from step-ca to the caller (including private key). Caller must handle private key securely.
        writeJSON(w, resp)
    }).Methods(http.MethodPost)

    // Revoke certificate (proxy to step /revoke). Expects JSON { serial, secret }
    api.HandleFunc("/certificates/revoke", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        var payload struct {
            Serial string `json:"serial"`
            Secret string `json:"secret"`
        }
        if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
            httpError(w, http.StatusBadRequest, "invalid request body")
            return
        }
        if payload.Serial == "" {
            httpError(w, http.StatusBadRequest, "serial required")
            return
        }
        if _, err := stepClient.RevokeCertificate(ctx, payload.Serial, payload.Secret); err != nil {
            logger.Printf("error revoking certificate: %v", err)
            httpError(w, http.StatusBadGateway, "failed to revoke certificate")
            return
        }
        // Update DB status
        if err := database.RevokeCertificate(ctx, payload.Serial); err != nil {
            logger.Printf("warning: failed to update certificate status in DB: %v", err)
        }
        _ = database.LogAudit(ctx, "certificate.revoke", "system", fmt.Sprintf("revoked certificate serial=%s", payload.Serial), r.RemoteAddr)
        writeJSON(w, map[string]string{"result": "ok"})
    }).Methods(http.MethodPost)

    // Download package: redirect to step-ca download URL (absolute)
    api.HandleFunc("/certificates/{id}/download", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        id := mux.Vars(r)["id"]
        if id == "" {
            httpError(w, http.StatusBadRequest, "id required")
            return
        }
        url, err := stepClient.DownloadCertificatePackage(ctx, id)
        if err != nil {
            logger.Printf("error building download url: %v", err)
            httpError(w, http.StatusInternalServerError, "failed to build download url")
            return
        }
        // Redirect client to step-ca download URL
        http.Redirect(w, r, url, http.StatusFound)
    }).Methods(http.MethodGet)

    // Start HTTP server with graceful shutdown
    srv := &http.Server{
        Addr:    listen,
        Handler: r,
    }

    // Listen for signals
    idleConnsClosed := make(chan struct{})
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        sig := <-sigCh
        logger.Printf("received signal %v, shutting down", sig)

        // Give server 10s to shutdown gracefully
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        if err := srv.Shutdown(ctx); err != nil {
            logger.Printf("HTTP server Shutdown: %v", err)
        }
        close(idleConnsClosed)
    }()

    logger.Printf("starting server on %s", listen)
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.Fatalf("HTTP server ListenAndServe: %v", err)
    }

    <-idleConnsClosed
    logger.Printf("server stopped")
}

// writeJSON writes v as JSON with 200 status.
func writeJSON(w http.ResponseWriter, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(v)
}

// httpError writes a JSON error response.
func httpError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// parseInt helper
func parseInt(s string) (int, error) {
    var i int
    _, err := fmt.Sscanf(s, "%d", &i)
    return i, err
}

// parseTimeOrNow parses RFC3339 time string or returns time.Now().UTC() if parsing fails or empty.
func parseTimeOrNow(s string) time.Time {
    if s == "" {
        return time.Now().UTC()
    }
    if t, err := time.Parse(time.RFC3339, s); err == nil {
        return t.UTC()
    }
    // try generic parse
    if t, err := time.Parse("2006-01-02T15:04:05Z07:00", s); err == nil {
        return t.UTC()
    }
    return time.Now().UTC()
}
