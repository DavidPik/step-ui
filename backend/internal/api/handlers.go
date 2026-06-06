package api

import (
    "net/http"
    "regexp"
    "strings"
    "time"

    "github.com/gin-gonic/gin"

    "github.com/DavidPik/step-ui/backend/internal/db"
    "github.com/DavidPik/step-ui/backend/internal/step"
)

//
// CA SETTINGS handlers (unchanged)
//

func GetCASettings(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        c.JSON(http.StatusOK, gin.H{
            "ca_url":           settings.CAURL,
            "root_fingerprint": settings.RootFingerprint,
            "provisioner_name": settings.ProvisionerName,
            "acme_directories": settings.ACMEDirectories,
            "status":           "ok",
        })
    }
}

func UpdateCASettings(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            CAURL             string   `json:"ca_url"`
            RootFingerprint   string   `json:"root_fingerprint"`
            ProvisionerName   string   `json:"provisioner_name"`
            ProvisionerSecret string   `json:"provisioner_secret"`
            ACMEDirectories   []string `json:"acme_directories"`
        }

        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        input.CAURL = strings.TrimSpace(input.CAURL)
        input.RootFingerprint = strings.TrimSpace(input.RootFingerprint)
        input.ProvisionerName = strings.TrimSpace(input.ProvisionerName)
        input.ProvisionerSecret = strings.TrimSpace(input.ProvisionerSecret)

        if input.CAURL == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "ca_url is required"})
            return
        }
        if !strings.HasPrefix(input.CAURL, "http://") && !strings.HasPrefix(input.CAURL, "https://") {
            c.JSON(http.StatusBadRequest, gin.H{"error": "ca_url must start with http:// or https://"})
            return
        }

        if input.RootFingerprint != "" {
            fpRegex := regexp.MustCompile(`^[A-Fa-f0-9:]{59,95}$`)
            if !fpRegex.MatchString(input.RootFingerprint) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid root fingerprint format"})
                return
            }
        }

        if input.ProvisionerName != "" {
            if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(input.ProvisionerName) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provisioner name"})
                return
            }
        }

        if input.ProvisionerSecret != "" && len(input.ProvisionerSecret) < 6 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "provisioner secret must be at least 6 characters"})
            return
        }

        for _, dir := range input.ACMEDirectories {
            if !strings.HasPrefix(dir, "http://") && !strings.HasPrefix(dir, "https://") {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ACME directory: " + dir})
                return
            }
        }

        settings := &db.CASettings{
            CAURL:             input.CAURL,
            RootFingerprint:   input.RootFingerprint,
            ProvisionerName:   input.ProvisionerName,
            ProvisionerSecret: input.ProvisionerSecret,
            ACMEDirectories:   input.ACMEDirectories,
        }

        if err := database.UpdateCASettings(settings); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update CA settings"})
            return
        }

        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "settings_updated",
            Details: "CA settings updated",
        })

        c.JSON(http.StatusOK, gin.H{"status": "updated"})
    }
}

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

        settings, _ := database.GetCASettings()
        selected := ""
        if settings != nil {
            selected = settings.ProvisionerName
        }

        items := make([]gin.H, 0, len(provs))
        for _, p := range provs {
            items = append(items, gin.H{
                "name":             p.Name,
                "type":             p.Type,
                "acme_directories": p.ACMEDirectories,
                "is_active":        p.Name == selected,
            })
        }

        c.JSON(http.StatusOK, gin.H{"items": items})
    }
}

func GetSelectedProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        settings, err := database.GetCASettings()
        if err != nil || settings == nil {
            c.JSON(http.StatusOK, gin.H{"name": ""})
            return
        }
        c.JSON(http.StatusOK, gin.H{"name": settings.ProvisionerName})
    }
}

func GetProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        name := c.Param("name")
        p, err := database.GetProvisionerByName(name)
        if err != nil || p == nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "provisioner not found"})
            return
        }

        // best-effort: fetch step-ca metadata (non-blocking)
        var meta interface{} = nil
        settings, _ := database.GetCASettings()
        if settings != nil && settings.CAURL != "" {
            client := step.NewClientFromSettings(settings)
            if provs, err := client.ListProvisioners(); err == nil {
                for _, sp := range provs {
                    if sp.Name == p.Name {
                        meta = sp
                        break
                    }
                }
            }
        }

        c.JSON(http.StatusOK, gin.H{
            "name":             p.Name,
            "type":             p.Type,
            "acme_directories": p.ACMEDirectories,
            "metadata":         meta,
        })
    }
}

func CreateProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            Name            string   `json:"name"`
            Type            string   `json:"type"`
            Secret          *string  `json:"secret"`
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

func SelectProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            Name   string  `json:"name"`
            Secret *string `json:"secret"`
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

        // Ensure provisioner exists in DB
        p, err := database.GetProvisionerByName(input.Name)
        if err != nil || p == nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "provisioner not found"})
            return
        }

        // If step-ca is configured and secret provided, we could attempt verification (best-effort).
        // We do not persist the secret.
        // For now accept the secret and set selected provisioner.
        settingsToUpdate, err := database.GetCASettings()
        if err != nil || settingsToUpdate == nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }
        settingsToUpdate.ProvisionerName = input.Name
        if err := database.UpdateCASettings(settingsToUpdate); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set selected provisioner"})
            return
        }

        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "provisioner_selected",
            Details: "name=" + input.Name,
        })

        c.JSON(http.StatusOK, gin.H{"status": "selected", "name": input.Name})
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
