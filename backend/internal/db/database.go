package db

import (
    "fmt"
    "log"
    "os"
    "strings"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

// Database wrapper
type Database struct {
    DB *gorm.DB
}

// Initialize DB connection + migrations + default CA settings
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

    // Auto-create tables
    if err := db.AutoMigrate(&Certificate{}, &AuditEvent{}, &CASettings{}); err != nil {
        log.Fatalf("Failed to migrate database schema: %v", err)
    }

    // Ensure at least one CA settings record exists
    ensureDefaultCASettings(db)

    return &Database{DB: db}
}

//
// ------------------------------------------------------------
// CA SETTINGS
// ------------------------------------------------------------
//

// Create default CA settings from ENV if DB is empty
func ensureDefaultCASettings(db *gorm.DB) {
    var count int64
    db.Model(&CASettings{}).Count(&count)

    if count == 0 {
        acmeDirs := []string{}
        if env := os.Getenv("ACME_DIRECTORIES"); env != "" {
            acmeDirs = strings.Split(env, ",")
        }

        settings := CASettings{
            CAURL:           os.Getenv("CA_URL"),
            RootFingerprint: os.Getenv("CA_FINGERPRINT"),
            ProvisionerName: os.Getenv("CA_PROVISIONER"),
            ProvisionerSecret: os.Getenv("CA_PROVISIONER_PASSWORD"),
            ACMEDirectories: acmeDirs,
        }

        if err := db.Create(&settings).Error; err != nil {
            log.Fatalf("Failed to create default CA settings: %v", err)
        }

        log.Println("Created default CA settings in database")
    }
}

// Return the single CA settings record
func (d *Database) GetCASettings() (*CASettings, error) {
    var settings CASettings
    if err := d.DB.First(&settings).Error; err != nil {
        return nil, err
    }
    return &settings, nil
}

// Update CA settings
func (d *Database) UpdateCASettings(settings *CASettings) error {
    return d.DB.Save(settings).Error
}

// Create new CA settings (rarely used, but needed for UI)
func (d *Database) CreateCASettings(settings *CASettings) error {
    return d.DB.Create(settings).Error
}

// Delete CA settings (only if you support multiple profiles)
func (d *Database) DeleteCASettings(id uint) error {
    return d.DB.Delete(&CASettings{}, id).Error
}

//
// ------------------------------------------------------------
// CERTIFICATES
// ------------------------------------------------------------
//

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

//
// ------------------------------------------------------------
// AUDIT LOG
// ------------------------------------------------------------
//

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
