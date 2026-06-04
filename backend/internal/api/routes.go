package api

import (
    "github.com/gin-gonic/gin"
    "github.com/DavidPik/step-ui/backend/internal/db"
    "github.com/DavidPik/step-ui/backend/internal/api/handlers"
)

func RegisterRoutes(r *gin.Engine, database *db.Database) {
    api := r.Group("/api")

    // CA Settings
    api.GET("/settings/ca", handlers.GetCASettings(database))
    api.PUT("/settings/ca", handlers.UpdateCASettings(database))

    // Provisioners
    api.GET("/provisioners", handlers.ListProvisioners(database))
    api.POST("/provisioners/select", handlers.SelectProvisioner(database))

    // Certificates
    api.POST("/certificates/issue", handlers.IssueCertificate(database))
    api.POST("/certificates/revoke", handlers.RevokeCertificate(database))

    // Audit log
    api.GET("/audit", handlers.GetAuditEvents(database))
}
