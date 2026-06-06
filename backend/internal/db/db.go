package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/jmoiron/sqlx"
    _ "github.com/go-sql-driver/mysql"
)

// DB wraps sqlx.DB and provides helper methods.
type DB struct {
    conn *sqlx.DB
}

// Default connection pool settings (approved)
const (
    defaultMaxOpenConns    = 25
    defaultMaxIdleConns    = 5
    defaultConnMaxLifetime = 5 * time.Minute
)

// InitDB opens connection to MySQL using DSN and ensures schema exists.
// DSN example: user:password@tcp(host:3306)/dbname?parseTime=true&loc=UTC
func InitDB(ctx context.Context, dsn string) (*DB, error) {
    if dsn == "" {
        return nil, errors.New("dsn is required")
    }

    // Ensure parseTime and loc=UTC are present for proper time parsing
    if !strings.Contains(dsn, "parseTime=") {
        if strings.Contains(dsn, "?") {
            dsn += "&parseTime=true"
        } else {
            dsn += "?parseTime=true"
        }
    }
    if !strings.Contains(dsn, "loc=") {
        dsn += "&loc=UTC"
    }

    dbx, err := sqlx.Open("mysql", dsn)
    if err != nil {
        return nil, fmt.Errorf("sqlx.Open: %w", err)
    }

    // Set connection pool defaults
    dbx.SetMaxOpenConns(defaultMaxOpenConns)
    dbx.SetMaxIdleConns(defaultMaxIdleConns)
    dbx.SetConnMaxLifetime(defaultConnMaxLifetime)

    // Ping to verify connection (no retries by design)
    ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    if err := dbx.PingContext(ctxPing); err != nil {
        _ = dbx.Close()
        return nil, fmt.Errorf("db ping: %w", err)
    }

    d := &DB{conn: dbx}

    // Ensure schema exists (create tables if DB is empty)
    if err := d.ensureSchema(ctx); err != nil {
        _ = dbx.Close()
        return nil, fmt.Errorf("ensureSchema: %w", err)
    }

    return d, nil
}

// Close closes the underlying DB connection.
func (d *DB) Close() error {
    if d == nil || d.conn == nil {
        return nil
    }
    return d.conn.Close()
}

// ensureSchema checks for presence of required tables and creates them if missing.
// It will also insert initial CA settings row if none exists.
func (d *DB) ensureSchema(ctx context.Context) error {
    // We'll check for one known table; if missing, create all tables.
    const checkTable = "audit_events"

    var exists bool
    query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
    var cnt int
    if err := d.conn.GetContext(ctx, &cnt, query, checkTable); err != nil {
        return fmt.Errorf("schema check query: %w", err)
    }
    exists = cnt > 0
    if exists {
        return nil
    }

    // Create tables
    tx, err := d.conn.BeginTxx(ctx, &sql.TxOptions{})
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() {
        // If still active and not committed, rollback
        _ = tx.Rollback()
    }()

    // Use InnoDB, utf8mb4
    stmts := []string{
        // CA settings (single row)
        `CREATE TABLE IF NOT EXISTS ca_settings (
            id BIGINT AUTO_INCREMENT PRIMARY KEY,
            ca_url TEXT,
            root_fingerprint TEXT,
            provisioner_name VARCHAR(255),
            acme_directories TEXT,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,

        // Provisioners
        `CREATE TABLE IF NOT EXISTS provisioners (
            name VARCHAR(255) PRIMARY KEY,
            type VARCHAR(50) NOT NULL,
            acme_directories TEXT,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,

        // Certificates
        `CREATE TABLE IF NOT EXISTS certificates (
            id VARCHAR(128) PRIMARY KEY,
            common_name VARCHAR(1024) NOT NULL,
            dns_names TEXT,
            serial VARCHAR(255) NOT NULL,
            not_before TIMESTAMP NOT NULL,
            not_after TIMESTAMP NOT NULL,
            certificate_pem MEDIUMTEXT,
            ca_chain_pem MEDIUMTEXT,
            status VARCHAR(50) NOT NULL DEFAULT 'active',
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            revoked_at TIMESTAMP NULL,
            INDEX idx_cert_serial (serial),
            INDEX idx_cert_common_name (common_name)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,

        // Audit events
        `CREATE TABLE IF NOT EXISTS audit_events (
            id BIGINT AUTO_INCREMENT PRIMARY KEY,
            timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            action VARCHAR(255) NOT NULL,
            user VARCHAR(255),
            details TEXT,
            ip VARCHAR(100)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
    }

    for _, s := range stmts {
        if _, err := tx.ExecContext(ctx, s); err != nil {
            return fmt.Errorf("create table exec: %w", err)
        }
    }

    // Insert initial CA settings row if none exists
    var settingsCount int
    if err := tx.GetContext(ctx, &settingsCount, "SELECT COUNT(*) FROM ca_settings"); err != nil {
        return fmt.Errorf("count ca_settings: %w", err)
    }
    if settingsCount == 0 {
        _, err := tx.ExecContext(ctx, `INSERT INTO ca_settings (ca_url, root_fingerprint, provisioner_name, acme_directories) VALUES (?, ?, ?, ?)`,
            "", "", "", "[]")
        if err != nil {
            return fmt.Errorf("insert initial ca_settings: %w", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit schema tx: %w", err)
    }

    return nil
}

// -----------------------------
// CA Settings
// -----------------------------

// GetCASettings returns the single CA settings row.
func (d *DB) GetCASettings(ctx context.Context) (*CASettings, error) {
    var s CASettings
    err := d.conn.GetContext(ctx, &s, "SELECT * FROM ca_settings LIMIT 1")
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return &s, nil
}

// UpdateCASettings updates the CA settings row (by id).
func (d *DB) UpdateCASettings(ctx context.Context, s *CASettings) error {
    if s == nil {
        return errors.New("nil settings")
    }
    _, err := d.conn.ExecContext(ctx, `UPDATE ca_settings SET ca_url = ?, root_fingerprint = ?, provisioner_name = ?, acme_directories = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
        s.CAURL, s.RootFingerprint, s.ProvisionerName, s.ACMEDirectories, s.ID)
    return err
}

// -----------------------------
// Provisioners
// -----------------------------

func (d *DB) ListProvisioners(ctx context.Context) ([]Provisioner, error) {
    var out []Provisioner
    if err := d.conn.SelectContext(ctx, &out, "SELECT name, type, acme_directories, created_at FROM provisioners ORDER BY name"); err != nil {
        return nil, err
    }
    return out, nil
}

func (d *DB) GetProvisioner(ctx context.Context, name string) (*Provisioner, error) {
    var p Provisioner
    if err := d.conn.GetContext(ctx, &p, "SELECT name, type, acme_directories, created_at FROM provisioners WHERE name = ? LIMIT 1", name); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return &p, nil
}

func (d *DB) CreateProvisioner(ctx context.Context, p *Provisioner) error {
    if p == nil {
        return errors.New("nil provisioner")
    }
    _, err := d.conn.ExecContext(ctx, `INSERT INTO provisioners (name, type, acme_directories) VALUES (?, ?, ?)`,
        p.Name, p.Type, p.ACMEDirectories)
    return err
}

func (d *DB) DeleteProvisioner(ctx context.Context, name string) error {
    // Physical delete as requested. Audit should be logged by caller via LogAudit.
    _, err := d.conn.ExecContext(ctx, `DELETE FROM provisioners WHERE name = ?`, name)
    return err
}

// -----------------------------
// Certificates
// -----------------------------

func (d *DB) ListCertificates(ctx context.Context) ([]Certificate, error) {
    var out []Certificate
    if err := d.conn.SelectContext(ctx, &out, "SELECT id, common_name, dns_names, serial, not_before, not_after, certificate_pem, ca_chain_pem, status, created_at, revoked_at FROM certificates ORDER BY created_at DESC"); err != nil {
        return nil, err
    }
    return out, nil
}

func (d *DB) GetCertificate(ctx context.Context, id string) (*Certificate, error) {
    var c Certificate
    if err := d.conn.GetContext(ctx, &c, "SELECT id, common_name, dns_names, serial, not_before, not_after, certificate_pem, ca_chain_pem, status, created_at, revoked_at FROM certificates WHERE id = ? LIMIT 1", id); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return &c, nil
}

// CreateCertificate inserts a certificate record. Note: private key must NOT be stored.
func (d *DB) CreateCertificate(ctx context.Context, c *Certificate) error {
    if c == nil {
        return errors.New("nil certificate")
    }
    _, err := d.conn.ExecContext(ctx, `INSERT INTO certificates (id, common_name, dns_names, serial, not_before, not_after, certificate_pem, ca_chain_pem, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
        c.ID, c.CommonName, c.DNSNames, c.Serial, c.NotBefore.UTC(), c.NotAfter.UTC(), c.CertificatePEM, c.CAChainPEM, c.Status)
    return err
}

// RevokeCertificate marks certificate as revoked and sets revoked_at timestamp.
func (d *DB) RevokeCertificate(ctx context.Context, serial string) error {
    _, err := d.conn.ExecContext(ctx, `UPDATE certificates SET status = 'revoked', revoked_at = CURRENT_TIMESTAMP WHERE serial = ?`, serial)
    return err
}

// DeleteCertificate physically deletes a certificate by id.
func (d *DB) DeleteCertificate(ctx context.Context, id string) error {
    _, err := d.conn.ExecContext(ctx, `DELETE FROM certificates WHERE id = ?`, id)
    return err
}

// -----------------------------
// Audit logging
// -----------------------------

// LogAudit inserts an audit event. This helper should be used by handlers to record actions.
func (d *DB) LogAudit(ctx context.Context, action, user, details, ip string) error {
    _, err := d.conn.ExecContext(ctx, `INSERT INTO audit_events (action, user, details, ip) VALUES (?, ?, ?, ?)`,
        action, user, details, ip)
    return err
}
