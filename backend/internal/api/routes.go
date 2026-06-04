package api

import (
    "github.com/gin-gonic/gin"
    "github.com/DavidPik/step-ui/backend/internal/db"
)

func RegisterRoutes(r *gin.Engine, database *db.Database) {
    api := r.Group("/api")

    // CA Settings
    api.GET("/settings", GetCASettings(database))
    api.PUT("/settings", UpdateCASettings(database))

    // Provisioners
    api.GET("/provisioners", ListProvisioners(database))
    api.POST("/provisioners/select", SelectProvisioner(database))
    api.GET("/provisioners/selected", GetSelectedProvisioner(database))

    // Certificates
    api.GET("/certificates", ListCertificates(database))
    api.GET("/certificates/:id", GetCertificate(database))
    api.GET("/certificates/:id/download", DownloadCertificatePackage(database))
    api.POST("/certificates/issue", IssueCertificate(database))
    api.POST("/certificates/revoke", RevokeCertificate(database))

    // Audit log
    api.GET("/audit", GetAuditEvents(database))
}
