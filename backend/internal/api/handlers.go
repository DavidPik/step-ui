package api

import (
    "net/http"
    "regexp"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-jose/go-jose/v3"

    "github.com/DavidPik/step-ui/backend/internal/db"
    "github.com/DavidPik/step-ui/backend/internal/step"
)

//
// PROVISIONERS handlers
//

func ListProvisioners(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        provs, err := database.ListProvisioners()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load provisioners"})
            return
        }

        items := make([]gin.H, 0, len(provs))
        for _, p := range provs {
            items = append(items, gin.H{
                "name":             p.Name,
                "type":             p.Type,
                "acme_directories": p.ACMEDirectories,
            })
        }

        c.JSON(http.StatusOK, gin.H{"items": items})
    }
}

func CreateProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            Name            string   `json:"name"`
            Type            string   `json:"type"`
            Secret          *string  `json:"secret"`
            JWK             string   `json:"jwk"`
            ACMEDirectories []string `json:"acme_directories"`
        }
        
        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        input.Name = strings.TrimSpace(input.Name)
        if input.Name == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
            return
        }
        if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(input.Name) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid name"})
            return
        }

        for _, d := range input.ACMEDirectories {
            if d != "" && !strings.HasPrefix(d, "http://") && !strings.HasPrefix(d, "https://") {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ACME directory: " + d})
                return
            }
        }

        // If step-ca is configured, attempt to create provisioner there using provided secret (best-effort).
        settings, _ := database.GetCASettings()
        if settings != nil && settings.CAURL != "" {
            client := step.NewClientFromSettings(settings)
            payload := map[string]interface{}{
                "name": input.Name,
                "type": input.Type,
            }
            if input.Secret != nil && *input.Secret != "" {
                payload["secret"] = *input.Secret
            }
            if err := client.CreateProvisioner(payload); err != nil {
                c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create provisioner in step-ca: " + err.Error()})
                return
            }
        }

        p := &db.Provisioner{
            Name:            input.Name,
            Type:            input.Type,
            JWK:             input.JWK,
            ACMEDirectories: input.ACMEDirectories,
        }

        if _, err := database.CreateProvisioner(p); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store provisioner"})
            return
        }

        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "provisioner_created",
            Details: "name=" + input.Name,
        })

        c.JSON(http.StatusCreated, gin.H{"status": "created", "name": input.Name})
    }
}

func UpdateProvisioner(database *db.Database) gin.HandlerFunc {
    // Per project decision: provisioners are immutable except create/delete/detail/select.
    // Return Method Not Allowed for update attempts.
    return func(c *gin.Context) {
        c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "provisioner update is not supported"})
    }
}

func DeleteProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        name := c.Param("name")

        // Expect secret in body for authorization (frontend will provide it).
        var input struct {
            Secret *string `json:"secret"`
        }
        _ = c.ShouldBindJSON(&input) // ignore error; we only check presence if step-ca requires it

        settings, _ := database.GetCASettings()
        if settings != nil && settings.CAURL != "" {
            client := step.NewClientFromSettings(settings)
            // We do not persist secret; if step-ca requires secret for delete, frontend must provide it
            // but StepClient.DeleteProvisioner currently does not accept secret; this is best-effort.
            _ = client.DeleteProvisioner(name)
        }

        if err := database.DeleteProvisioner(name); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete provisioner"})
            return
        }

        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "provisioner_deleted",
            Details: "name=" + name,
        })

        c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
    }
}

func GetProvisionerStatuses(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        provs, err := database.ListProvisioners()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load provisioners"})
            return
        }

        items := make([]gin.H, 0, len(provs))
        for _, p := range provs {
            status := CheckProvisionerStatus(&p)
            items = append(items, gin.H{
                "name":   p.Name,
                "status": status,
            })
        }

        c.JSON(http.StatusOK, gin.H{"items": items})
    }
}

func GetProvisionerStatus(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        name := c.Param("name")

        p, err := database.GetProvisionerByName(name)
        if err != nil || p == nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "provisioner not found"})
            return
        }

        status := CheckProvisionerStatus(p)

        c.JSON(http.StatusOK, gin.H{
            "name":   p.Name,
            "status": status,
        })
    }
}

//
// CERTIFICATES handlers (unchanged from previous implementation)
//

func IssueCertificate(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req step.CertificateRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        if strings.TrimSpace(req.CommonName) == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "common_name is required"})
            return
        }

        if len(req.DNSNames) > 20 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "too many DNS names"})
            return
        }
        for _, dns := range req.DNSNames {
            if !isValidDNSName(dns) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid DNS name: " + dns})
                return
            }
        }

        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        client := step.NewClientFromSettings(settings)

        resp, err := client.IssueCertificate(req)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        meta, err := step.ParseCertificateMetadata(resp.Certificate)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse certificate metadata"})
            return
        }

        cert := &db.Certificate{
            CommonName:     req.CommonName,
            DNSNames:       strings.Join(req.DNSNames, ","),
            Serial:         meta.Serial,
            NotBefore:      meta.NotBefore,
            NotAfter:       meta.NotAfter,
            CertificatePEM: resp.Certificate,
            PrivateKeyPEM:  resp.PrivateKey,
            CAChainPEM:     resp.CABundle,
        }

        id, err := database.CreateCertificate(cert)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store certificate"})
            return
        }

        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "certificate_issued",
            Details: "CN=" + req.CommonName,
        })

        c.JSON(http.StatusOK, gin.H{
            "status":      "issued",
            "id":          id,
            "common_name": req.CommonName,
            "serial":      meta.Serial,
            "not_before":  meta.NotBefore,
            "not_after":   meta.NotAfter,
            "certificate": resp.Certificate,
            "private_key": resp.PrivateKey,
            "ca_bundle":   resp.CABundle,
        })
    }
}

func RevokeCertificate(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            Serial string `json:"serial"`
        }

        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        input.Serial = strings.TrimSpace(input.Serial)

        if input.Serial == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "serial is required"})
            return
        }

        if !regexp.MustCompile(`^[0-9A-Fa-f]+$`).MatchString(input.Serial) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid serial format"})
            return
        }

        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        client := step.NewClientFromSettings(settings)

        if err := client.RevokeCertificate(input.Serial); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "certificate_revoked",
            Details: "Serial=" + input.Serial,
        })

        c.JSON(http.StatusOK, gin.H{"status": "revoked"})
    }
}

func ListCertificates(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        certs, err := database.ListCertificates()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load certificates"})
            return
        }

        c.JSON(http.StatusOK, gin.H{
            "items": certs,
        })
    }
}

func GetCertificate(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.Param("id")

        cert, err := database.GetCertificateByID(id)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
            return
        }

        c.JSON(http.StatusOK, cert)
    }
}

func DownloadCertificatePackage(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.Param("id")

        cert, err := database.GetCertificateByID(id)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
            return
        }

        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        client := step.NewClientFromSettings(settings)

        resp := step.CertificateResponse{
            Certificate: cert.CertificatePEM,
            PrivateKey:  cert.PrivateKeyPEM,
            CABundle:    cert.CAChainPEM,
        }

        zipBytes, err := client.BuildCertificatePackage(cert.CommonName, &resp)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build certificate package"})
            return
        }

        c.Header("Content-Type", "application/zip")
        c.Header("Content-Disposition", "attachment; filename=\""+cert.CommonName+".zip\"")
        c.Data(http.StatusOK, "application/zip", zipBytes)
    }
}

//
// HELPERS
//

func isValidDNSName(s string) bool {
    s = strings.TrimSpace(s)
    if s == "" {
        return false
    }

    if !regexp.MustCompile(`^[a-zA-Z0-9.-]+$`).MatchString(s) {
        return false
    }
    if !strings.Contains(s, ".") {
        return false
    }
    return true
}

func CheckProvisionerStatus(p *db.Provisioner) string {
    if p.Type == "ACME" {
        return checkACMEStatus(p)
    }
    if p.Type == "JWK" {
        return checkJWKStatus(p)
    }
    return "unknown"
}

func checkACMEStatus(p *db.Provisioner) string {
    if len(p.ACMEDirectories) == 0 {
        return "unknown"
    }

    client := http.Client{Timeout: 3 * time.Second}

    for _, dir := range p.ACMEDirectories {
        resp, err := client.Get(dir)
        if err != nil {
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode == 200 {
            return "online"
        }
        return "error"
    }

    return "offline"
}

func checkJWKStatus(p *db.Provisioner) string {
    if p.JWK == "" {
        return "unknown"
    }

    _, err := jose.ParseJWK(p.JWK)
    if err != nil {
        return "error"
    }

    return "online"
}
