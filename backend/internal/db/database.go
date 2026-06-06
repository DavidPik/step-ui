package db

import (
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

// Database wraps gorm DB
type Database struct {
    DB *gorm.DB
}

// NewDatabase initializes DB connection and migrates schema.
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

    dbConn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatalf("Failed to connect to MariaDB: %v", err)
    }

    // Auto-migrate all models
    if err := dbConn.AutoMigrate(&Certificate{}, &AuditEvent{}, &CASettings{}, &Provisioner{}); err != nil {
        log.Fatalf("Failed to migrate database schema: %v", err)
    }

    ensureDefaultCASettings(dbConn)

    return &Database{DB: dbConn}
}

// ensureDefaultCASettings creates a single CASettings row from environment if none exists.
func ensureDefaultCASettings(db *gorm.DB) {
    var count int64
    db.Model(&CASettings{}).Count(&count)

    if count == 0 {
        acmeDirs := []string{}
        if env := os.Getenv("ACME_DIRECTORIES"); env != "" {
            acmeDirs = strings.Split(env, ",")
        }

        settings := CASettings{
            CAURL:             os.Getenv("CA_URL"),
            RootFingerprint:   os.Getenv("CA_FINGERPRINT"),
            ProvisionerName:   os.Getenv("CA_PROVISIONER"),
            ProvisionerSecret: os.Getenv("CA_PROVISIONER_PASSWORD"),
            ACMEDirectories:   acmeDirs,
        }

        if err := db.Create(&settings).Error; err != nil {
            log.Fatalf("Failed to create default CA settings: %v", err)
        }

        log.Println("Created default CA settings in database")
    }
}

// -----------------------------
// CA SETTINGS CRUD
// -----------------------------

// GetCASettings returns the first CASettings row.
func (d *Database) GetCASettings() (*CASettings, error) {
    var settings CASettings
    if err := d.DB.First(&settings).Error; err != nil {
        return nil, err
    }
    return &settings, nil
}

// UpdateCASettings saves CASettings (upsert behavior via Save).
func (d *Database) UpdateCASettings(settings *CASettings) error {
    return d.DB.Save(settings).Error
}

// SetSelectedProvisioner sets the active provisioner name in CASettings.
func (d *Database) SetSelectedProvisioner(name string) error {
    settings, err := d.GetCASettings()
    if err != nil {
        return err
    }
    settings.ProvisionerName = name
    return d.UpdateCASettings(settings)
}

// -----------------------------
// CERTIFICATES CRUD
// -----------------------------

// CreateCertificate stores certificate and returns its ID.
func (d *Database) CreateCertificate(cert *Certificate) (uint, error) {
    if err := d.DB.Create(cert).Error; err != nil {
        return 0, err
    }
    return cert.ID, nil
}

func (d *Database) GetCertificateByID(id string) (*Certificate, error) {
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

// -----------------------------
// PROVISIONERS CRUD
// -----------------------------

// CreateProvisioner stores provisioner metadata. Secrets are not persisted.
func (d *Database) CreateProvisioner(p *Provisioner) (uint, error) {
    if err := d.DB.Create(p).Error; err != nil {
        return 0, err
    }
    return p.ID, nil
}

// GetProvisionerByName returns provisioner by name.
func (d *Database) GetProvisionerByName(name string) (*Provisioner, error) {
    var p Provisioner
    if err := d.DB.Where("name = ?", name).First(&p).Error; err != nil {
        return nil, err
    }
    return &p, nil
}

// ListProvisioners returns all provisioners ordered by newest first.
func (d *Database) ListProvisioners() ([]Provisioner, error) {
    var provs []Provisioner
    if err := d.DB.Order("id desc").Find(&provs).Error; err != nil {
        return nil, err
    }
    return provs, nil
}

// UpdateProvisioner updates provisioner metadata. Per project decision provisioners are immutable,
// but this method allows safe metadata updates if ever needed.
func (d *Database) UpdateProvisioner(oldName string, p *Provisioner) error {
    return d.DB.Model(&Provisioner{}).Where("name = ?", oldName).Updates(map[string]interface{}{
        "name":             p.Name,
        "acme_directories": p.ACMEDirectoriesCSV,
        "type":             p.Type,
    }).Error
}

// DeleteProvisioner removes provisioner by name.
func (d *Database) DeleteProvisioner(name string) error {
    return d.DB.Where("name = ?", name).Delete(&Provisioner{}).Error
}

// -----------------------------
// AUDIT LOG
// -----------------------------

func (d *Database) LogAuditEvent(event *AuditEvent) error {
    return d.DB.Create(event).Error
}

func (d *Database) QueryAuditEvents(
    from, to *time.Time,
    action, user string,
) ([]AuditEvent, error) {

    query := d.DB.Model(&AuditEvent{})

    if from != nil {
        query = query.Where("created_at >= ?", *from)
    }
    if to != nil {
        query = query.Where("created_at <= ?", *to)
    }
    if action != "" {
        query = query.Where("action = ?", action)
    }
    if user != "" {
        query = query.Where("user = ?", user)
    }

    var events []AuditEvent
    if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
        return nil, err
    }

    return events, nil
}
