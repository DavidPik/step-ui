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
// ------------------------------------------------------------
// CA SETTINGS
// ------------------------------------------------------------
//

func GetCASettings(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        settings, err := database.GetCASettings()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load CA settings"})
            return
        }

        // Frontend NECHCE provisioner secret
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
            CAURL           string   `json:"ca_url"`
            RootFingerprint string   `json:"root_fingerprint"`
            ProvisionerName string   `json:"provisioner_name"`
            ProvisionerSecret string `json:"provisioner_secret"`
            ACMEDirectories []string `json:"acme_directories"`
        }

        if err := c.ShouldBindJSON(&input); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        // Trim
        input.CAURL = strings.TrimSpace(input.CAURL)
        input.RootFingerprint = strings.TrimSpace(input.RootFingerprint)
        input.ProvisionerName = strings.TrimSpace(input.ProvisionerName)
        input.ProvisionerSecret = strings.TrimSpace(input.ProvisionerSecret)

        // Validate CA URL
        if input.CAURL == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "ca_url is required"})
            return
        }
        if !strings.HasPrefix(input.CAURL, "http://") && !strings.HasPrefix(input.CAURL, "https://") {
            c.JSON(http.StatusBadRequest, gin.H{"error": "ca_url must start with http:// or https://"})
            return
        }

        // Validate fingerprint (SHA256:XX:XX:XX…)
        if input.RootFingerprint != "" {
            fpRegex := regexp.MustCompile(`^[A-Fa-f0-9:]{59,95}$`)
            if !fpRegex.MatchString(input.RootFingerprint) {
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

        // Validate secret
        if input.ProvisionerSecret != "" && len(input.ProvisionerSecret) < 6 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "provisioner secret must be at least 6 characters"})
            return
        }

        // Validate ACME directories
        for _, dir := range input.ACMEDirectories {
            if !strings.HasPrefix(dir, "http://") && !strings.HasPrefix(dir, "https://") {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ACME directory: " + dir})
                return
            }
        }

        // Save
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

        // Audit
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "settings_updated",
            Details: "CA settings updated",
        })

        c.JSON(http.StatusOK, gin.H{"status": "updated"})
    }
}

//
// ------------------------------------------------------------
// CERTIFICATES — ISSUE, REVOKE, LIST, GET, DOWNLOAD
// ------------------------------------------------------------
//

func IssueCertificate(database *db.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req step.CertificateRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
            return
        }

        // Validate CN
        if strings.TrimSpace(req.CommonName) == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "common_name is required"})
            return
        }

        // Validate DNS names
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

        // Call step-ca
        resp, err := client.IssueCertificate(req)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        // Parse metadata
        meta, err := step.ParseCertificateMetadata(resp.Certificate)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse certificate metadata"})
            return
        }

        // Save to DB
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

        // Audit
        _ = database.LogAuditEvent(&db.AuditEvent{
            User:    "system",
            Action:  "certificate_issued",
            Details: "CN=" + req.CommonName,
        })

        // Return JSON (NOT ZIP)
        c.JSON(http.StatusOK, gin.H{
            "status":        "issued",
            "id":            id,
            "common_name":   req.CommonName,
            "serial":        meta.Serial,
            "not_before":    meta.NotBefore,
            "not_after":     meta.NotAfter,
            "certificate":   resp.Certificate,
            "private_key":   resp.PrivateKey,
            "ca_bundle":     resp.CABundle,
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

        // Audit
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
// ------------------------------------------------------------
// AUDIT LOG
// ------------------------------------------------------------
//

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

        c.JSON(http.StatusOK, gin.H{
            "items": events,
        })
    }
}

//
// ------------------------------------------------------------
// HELPERS
// ------------------------------------------------------------
//

func isValidDNSName(s string) bool {
    s = strings.TrimSpace(s)
    if s == "" {
        return false
    }

    // velmi jednoduchá validace: povolené znaky + aspoň jedna tečka
    if !regexp.MustCompile(`^[a-zA-Z0-9.-]+$`).MatchString(s) {
        return false
    }
    if !strings.Contains(s, ".") {
        return false
    }
    return true
}
