package api

import (
    "github.com/gin-gonic/gin"
    "github.com/DavidPik/step-ui/backend/internal/db"
)

func RegisterRoutes(r *gin.Engine, database *db.Database) {
    api := r.Group("/api")

    //
    // PROVISIONERS
    //
    api.GET("/provisioners", ListProvisioners(database))
    api.GET("/provisioners/:name", GetProvisioner(database))
    api.POST("/provisioners", CreateProvisioner(database))
    api.GET("/provisioners/status", GetProvisionerStatuses(database))
    api.GET("/provisioners/:name/status", GetProvisionerStatus(database))

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
