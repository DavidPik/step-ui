package api

import (
    "github.com/gin-gonic/gin"
    "github.com/DavidPik/step-ui/backend/internal/db"
)

func RegisterRoutes(r *gin.Engine, database *db.Database) {
    api := r.Group("/api")

    // Provisioners
    api.GET("/provisioners", func(c *gin.Context) {
        GetProvisioners(c, database)
    })

    // Certificates
    api.POST("/certificates/issue", func(c *gin.Context) {
        IssueCertificate(c, database)
    })

    api.POST("/certificates/revoke", func(c *gin.Context) {
        RevokeCertificate(c, database)
    })

    api.GET("/certificates", func(c *gin.Context) {
        ListCertificates(c, database)
    })

    api.GET("/certificates/:id", func(c *gin.Context) {
        GetCertificate(c, database)
    })

    api.GET("/certificates/:id/download", func(c *gin.Context) {
        DownloadCertificatePackage(c, database)
    })
}
