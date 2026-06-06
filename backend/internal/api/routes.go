package api

import (
    "github.com/gin-gonic/gin"
    "github.com/DavidPik/step-ui/backend/internal/db"
)

func RegisterRoutes(r *gin.Engine, database *db.Database) {
    api := r.Group("/api")

    //
    // CA SETTINGS
    //
    api.GET("/settings", GetCASettings(database))
    api.PUT("/settings", UpdateCASettings(database))

    //
    // PROVISIONERS
    //
    api.GET("/provisioners", ListProvisioners(database))
    api.GET("/provisioners/selected", GetSelectedProvisioner(database))
    api.GET("/provisioners/:name", GetProvisioner(database))
    api.POST("/provisioners", CreateProvisioner(database))
    api.DELETE("/provisioners/:name", DeleteProvisioner(database))
    api.POST("/provisioners/select", SelectProvisioner(database))

    //
    // CERTIFICATES
    //
    api.GET("/certificates", ListCertificates(database))
    api.GET("/certificates/:id", GetCertificate(database))
    api.POST("/certificates/issue", IssueCertificate(database))
    api.POST("/certificates/revoke", RevokeCertificate(database))
    api.GET("/certificates/:id/download", DownloadCertificatePackage(database))

    //
    // AUDIT LOG
    //
    api.GET("/audit", GetAuditEvents(database))
}
