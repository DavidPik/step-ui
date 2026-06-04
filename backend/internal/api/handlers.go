package handlers

import (
    "net/http"

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

        c.JSON(http.StatusOK, gin.H{"status": "provisioner selected"})
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

        c.JSON(http.StatusOK, gin.H{"status": "revoked"})
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

        var fromTime, toTime time.Time
        var err error

        // Parse "from"
        if fromStr != "" {
            fromTime, err = time.Parse(time.RFC3339, fromStr)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' timestamp, expected RFC3339"})
                return
            }
        }

        // Parse "to"
        if toStr != "" {
            toTime, err = time.Parse(time.RFC3339, toStr)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' timestamp, expected RFC3339"})
                return
            }
        }

        // Build query
        query := database.DB.Model(&db.AuditEvent{})

        if !fromTime.IsZero() {
            query = query.Where("created_at >= ?", fromTime)
        }
        if !toTime.IsZero() {
            query = query.Where("created_at <= ?", toTime)
        }
        if action != "" {
            query = query.Where("action = ?", action)
        }
        if user != "" {
            query = query.Where("user = ?", user)
        }

        var events []db.AuditEvent
        if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load audit log"})
            return
        }

        c.JSON(http.StatusOK, events)
    }
}
