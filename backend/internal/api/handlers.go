package api

import (
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"

    "github.com/DavidPik/step-ui/backend/internal/db"
    "github.com/DavidPik/step-ui/backend/internal/step"
)

// ------------------------------------------------------------
// CA SETTINGS
// ------------------------------------------------------------

func GetCASettings(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }
        c.JSON(http.StatusOK, settings)
    }
}

func UpdateCASettings(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input db.CASettings
        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        // Trim
        input.CAURL = strings.TrimSpace(input.CAURL)
        input.RootFingerprint = strings.TrimSpace(input.RootFingerprint)
        input.ProvisionerName = strings.TrimSpace(input.ProvisionerName)
        input.ProvisionerSecret = strings.TrimSpace(input.ProvisionerSecret)

        // Validate CAURL
        if input.CAURL == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "CAURL is required"})
            return
        }
        if !strings.HasPrefix(input.CAURL, "http://") && !strings.HasPrefix(input.CAURL, "https://") {
            c.JSON(http.StatusBadRequest, gin.H{"error": "CAURL must start with http:// or https://"})
            return
        }

        // Validate fingerprint
        if input.RootFingerprint != "" {
            if !regexp.MustCompile(`^[A-Fa-f0-9]{40,64}$`).MatchString(input.RootFingerprint) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid root fingerprint format"})
                return
            }
        }

        // Validate provisioner name
        if input.ProvisionerName != "" {
            if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(input.ProvisionerName) {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provisioner name"})
                return
            }
        }

        // Validate provisioner secret
        if input.ProvisionerSecret != "" && len(input.ProvisionerSecret) < 6 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "provisioner secret must be at least 6 characters"})
            return
        }

        // Validate ACME directories
        for _, dir := range input.ACMEDirectories {
            if !strings.HasPrefix(dir, "http://") && !strings.HasPrefix(dir, "https://") {
                c.JSON(http.StatusBadRequest, gin.H{"error": "ACME directory must be a valid URL: " + dir})
                return
            }
        }

        // Save Settings
        if err := database.UpdateCASettings(&input); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update CA settings"})
            return
        }

        // Audit log
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "settings_updated",
            Details: "CA settings updated",
        })

        c.JSON(http.StatusOK, gin.H{"status": "updated"})
    }
}

// ------------------------------------------------------------
// PROVISIONERS
// ------------------------------------------------------------

func ListProvisioners(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        client := step.NewClientFromSettings(settings)
        provisioners, err := client.ListProvisioners()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusOK, provisioners)
    }
}

func SelectProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            Name   string `json:"name"`
            Secret string `json:"secret"`
        }

        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        // Validate Name & Secret
        if input.Name == "" || input.Secret == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "name and secret are required"})
            return
        }

        // Validate Secret
        if len(input.Secret) < 6 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "secret must be at least 6 characters"})
            return
        }

        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        settings.ProvisionerName = input.Name
        settings.ProvisionerSecret = input.Secret

        if err := database.UpdateCASettings(settings); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update provisioner"})
            return
        }

        // Audit log
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "provisioner_selected",
            Details: "Provisioner: " + input.Name,
        })

        c.JSON(http.StatusOK, gin.H{"status": "provisioner selected"})
    }
}

func CreateProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var input struct {
            Name   string `json:"name"`
            Type   string `json:"type"`
            Secret string `json:"secret"`
        }

        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        // Trim whitespace
        input.Name = strings.TrimSpace(input.Name)
        input.Type = strings.TrimSpace(input.Type)
        input.Secret = strings.TrimSpace(input.Secret)

        // Validate name
        if input.Name == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
            return
        }
        if len(input.Name) > 64 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "name too long"})
            return
        }
        if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(input.Name) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "name contains invalid characters"})
            return
        }

        // Validate type
        validTypes := map[string]bool{
            "JWK":    true,
            "ACME":   true,
            "SSHPOP": true,
        }
        if !validTypes[input.Type] {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provisioner type"})
            return
        }

        // Validate secret (only for JWK)
        if input.Type == "JWK" {
            if input.Secret == "" {
                c.JSON(http.StatusBadRequest, gin.H{"error": "secret is required for JWK provisioner"})
                return
            }
            if len(input.Secret) < 6 {
                c.JSON(http.StatusBadRequest, gin.H{"error": "secret must be at least 6 characters"})
                return
            }
        }

        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        client := step.NewClientFromSettings(settings)

        // Prepare payload
        payload := map[string]interface{}{
            "name": input.Name,
            "type": input.Type,
        }
        if input.Type == "JWK" {
            payload["password"] = input.Secret
        }

        // Call client
        if err := client.CreateProvisioner(payload); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        // Audit log
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "provisioner_created",
            Details: "Provisioner: " + input.Name,
        })

        c.JSON(http.StatusOK, gin.H{"status": "created"})
    }
}

func DeleteProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        name := strings.TrimSpace(c.Param("name"))

        // Validate name
        if name == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
            return
        }

        if !regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(name) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provisioner name"})
            return
        }

        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        // Prevent deleting active provisioner
        if settings.ProvisionerName == name {
            c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete active provisioner"})
            return
        }

        client := step.NewClientFromSettings(settings)

        // Call client
        if err := client.DeleteProvisioner(name); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        // Audit log
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "provisioner_deleted",
            Details: "Provisioner: " + name,
        })

        c.JSON(http.StatusOK, gin.H{"status": "deleted"})
    }
}

// ------------------------------------------------------------
// PROVISIONER STATUS
// ------------------------------------------------------------

func GetSelectedProvisioner(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        c.JSON(http.StatusOK, gin.H{
            "name":   settings.ProvisionerName,
            "secret": settings.ProvisionerSecret,
        })
    }
}

// ------------------------------------------------------------
// CERTIFICATES
// ------------------------------------------------------------

func IssueCertificate(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req step.CertificateRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        // Valiate CommonName
        if req.CommonName == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "common_name is required"})
            return
        }

        // Validate DNSName
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

        // Issue certificate
        resp, err := client.IssueCertificate(req)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        // Parse & Save certificate for metadata
        certMeta, err := step.ParseCertificateMetadata(resp.Certificate)
        if err == nil {
            _ = database.CreateCertificate(&db.Certificate{
                CommonName:     req.CommonName,
                DNSNames:       strings.Join(req.DNSNames, ","),
                Serial:         certMeta.Serial,
                NotBefore:      certMeta.NotBefore,
                NotAfter:       certMeta.NotAfter,
                CertificatePEM: resp.Certificate,
                PrivateKeyPEM:  resp.PrivateKey,
                CAChainPEM:     resp.CABundle,
            })
        }

        // Audit log
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "certificate_issued",
            Details: "CN=" + req.CommonName,
        })

        // Build ZIP package
        zipBytes, err := client.BuildCertificatePackage(req.CommonName, resp)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build certificate package"})
            return
        }

        // Send ZIP as download
        c.Header("Content-Type", "application/zip")
        c.Header("Content-Disposition", "attachment; filename=\""+req.CommonName+".zip\"")
        c.Data(http.StatusOK, "application/zip", zipBytes)
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

        // Trim whitespaces
        input.Serial = strings.TrimSpace(input.Serial)

        // Validate Serial
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

        // Audit log
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "certificate_revoked",
            Details: "Serial=" + input.Serial,
        })

        c.JSON(http.StatusOK, gin.H{"status": "revoked"})
    }
}

// ------------------------------------------------------------
// CERTIFICATE LISTING
// ------------------------------------------------------------

func ListCertificates(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        certs, err := database.ListCertificates()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load certificates"})
            return
        }
        c.JSON(http.StatusOK, certs)
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

// ------------------------------------------------------------
// AUDIT LOG
// ------------------------------------------------------------

func GetAuditEvents(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        fromStr := c.Query("from")
        toStr := c.Query("to")
        action := c.Query("action")
        user := c.Query("user")

        var fromPtr, toPtr *time.Time

        if fromStr != "" {
            t, err := time.Parse(time.RFC3339, fromStr)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' timestamp"})
                return
            }
            fromPtr = &t
        }

        if toStr != "" {
            t, err := time.Parse(time.RFC3339, toStr)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' timestamp"})
                return
            }
            toPtr = &t
        }

        events, err := database.QueryAuditEvents(fromPtr, toPtr, action, user)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load audit log"})
            return
        }

        c.JSON(http.StatusOK, events)
    }
}
