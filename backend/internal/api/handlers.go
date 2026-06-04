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

        // Parse certificate for metadata
        certMeta, err := step.ParseCertificateMetadata(resp.Certificate)
        if err == nil {
            _ = database.CreateCertificate(&db.Certificate{
                CommonName: req.CommonName,
                DNSNames:   strings.Join(req.DNSNames, ","),
                Serial:     certMeta.Serial,
                NotBefore:  certMeta.NotBefore,
                NotAfter:   certMeta.NotAfter,
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

        // Build ZIP package again
        resp := step.IssueResponse{
            Certificate: cert.CertificatePEM,
            PrivateKey:  cert.PrivateKeyPEM,
            CAChain:     cert.CAChainPEM,
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
