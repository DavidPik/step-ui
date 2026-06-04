package db

import (
    "fmt"
    "log"
    "os"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

type Database struct {
    DB *gorm.DB
}

func NewDatabase() *Database {
    user := os.Getenv("DB_USER")
    pass := os.Getenv("DB_PASSWORD")
    host := os.Getenv("DB_HOST")
    port := os.Getenv("DB_PORT")
    name := os.Getenv("DB_NAME")

    if user == "" || pass == "" || host == "" || port == "" || name == "" {
        log.Fatalf("Missing database configuration: DB_USER, DB_PASSWORD, DB_HOST, DB_PORT, DB_NAME must be set")
    }

    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        user, pass, host, port, name)

    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatalf("Failed to connect to MariaDB: %v", err)
    }

    // Auto-create tables if missing
    if err := db.AutoMigrate(&Certificate{}, &AuditEvent{}); err != nil {
        log.Fatalf("Failed to migrate database schema: %v", err)
    }

    return &Database{DB: db}
}

// ------------------------------------------------------------
// Certificate CRUD
// ------------------------------------------------------------

func (d *Database) CreateCertificate(cert *Certificate) error {
    return d.DB.Create(cert).Error
}

func (d *Database) GetCertificate(id uint) (*Certificate, error) {
    var cert Certificate
    if err := d.DB.First(&cert, id).Error; err != nil {
        return nil, err
    }
    return &cert, nil
}

func (d *Database) ListCertificates() ([]Certificate, error) {
    var certs []Certificate
    if err := d.DB.Order("id desc").Find(&certs).Error; err != nil {
        return nil, err
    }
    return certs, nil
}

func (d *Database) UpdateCertificate(cert *Certificate) error {
    return d.DB.Save(cert).Error
}

func (d *Database) DeleteCertificate(id uint) error {
    return d.DB.Delete(&Certificate{}, id).Error
}

// ------------------------------------------------------------
// Audit log
// ------------------------------------------------------------

func (d *Database) LogAuditEvent(event *AuditEvent) error {
    return d.DB.Create(event).Error
}

func (d *Database) GetAuditEvents() ([]AuditEvent, error) {
    var events []AuditEvent
    if err := d.DB.Order("id desc").Find(&events).Error; err != nil {
        return nil, err
    }
    return events, nil
}
