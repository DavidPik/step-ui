package db

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "time"

    _ "github.com/go-sql-driver/mysql"
    "github.com/google/uuid"
)

// InitDB opens DB and ensures schema exists.
func InitDB(ctx context.Context, dsn string) (*Database, error) {
    // Expect DSN like: user:pass@tcp(host:3306)/dbname?parseTime=true&loc=UTC
    conn, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    // Configure pool for MySQL
    conn.SetMaxOpenConns(25)
    conn.SetMaxIdleConns(5)
    conn.SetConnMaxLifetime(5 * time.Minute)
    db := &Database{conn: conn}

    if err := db.ensureSchema(ctx); err != nil {
        conn.Close()
        return nil, err
    }
    return db, nil
}

func (db *Database) Close() error {
    return db.conn.Close()
}

func (db *Database) ensureSchema(ctx context.Context) error {
    // MySQL-compatible DDL. Use TEXT for JSON-like arrays for broad compatibility.
    schema := `
CREATE TABLE IF NOT EXISTS provisioners (
    name VARCHAR(255) PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    jwk TEXT,
    acme_directories TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS certificates (
    id VARCHAR(64) PRIMARY KEY,
    common_name VARCHAR(255) NOT NULL,
    dns_names TEXT,
    serial VARCHAR(255),
    not_before DATETIME,
    not_after DATETIME,
    certificate_pem MEDIUMTEXT,
    private_key_pem MEDIUMTEXT,
    ca_chain_pem MEDIUMTEXT,
    status VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_events (
    id INT AUTO_INCREMENT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action VARCHAR(255) NOT NULL,
    user VARCHAR(255) NOT NULL,
    details TEXT,
    ip VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS ca_settings (
    id INT PRIMARY KEY,
    provisioner_name VARCHAR(255),
    acme_directories TEXT,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT IGNORE INTO ca_settings (id, provisioner_name, acme_directories) VALUES (1, '', '[]');
`
    _, err := db.conn.ExecContext(ctx, schema)
    return err
}

// --- Helpers ---

func NewID() string {
    return uuid.New().String()
}

func marshalStringArray(a []string) (string, error) {
    if a == nil {
        return "[]", nil
    }
    b, err := json.Marshal(a)
    return string(b), err
}

func unmarshalStringArray(s sql.NullString) ([]string, error) {
    if !s.Valid || s.String == "" {
        return []string{}, nil
    }
    var out []string
    if err := json.Unmarshal([]byte(s.String), &out); err != nil {
        return nil, err
    }
    return out, nil
}

// -----------------------------
// Provisioners (context-aware)
// -----------------------------

func (db *Database) ListProvisioners(ctx context.Context) ([]Provisioner, error) {
    rows, err := db.conn.QueryContext(ctx, `
        SELECT name, type, jwk, acme_directories, created_at
        FROM provisioners
        ORDER BY name ASC
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []Provisioner

    for rows.Next() {
        var p Provisioner
        var dirsJSON sql.NullString
        var created sql.NullTime

        if err := rows.Scan(&p.Name, &p.Type, &p.JWK, &dirsJSON, &created); err != nil {
            return nil, err
        }

        p.ACMEDirectories, _ = unmarshalStringArray(dirsJSON)
        if created.Valid {
            p.CreatedAt = created.Time
        }

        out = append(out, p)
    }

    return out, nil
}

func (db *Database) GetProvisioner(ctx context.Context, name string) (*Provisioner, error) {
    row := db.conn.QueryRowContext(ctx, `
        SELECT name, type, jwk, acme_directories, created_at
        FROM provisioners
        WHERE name = ?
    `, name)

    var p Provisioner
    var dirsJSON sql.NullString
    var created sql.NullTime

    err := row.Scan(&p.Name, &p.Type, &p.JWK, &dirsJSON, &created)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    p.ACMEDirectories, _ = unmarshalStringArray(dirsJSON)
    if created.Valid {
        p.CreatedAt = created.Time
    }

    return &p, nil
}

func (db *Database) CreateProvisioner(ctx context.Context, p *Provisioner) error {
    dirsJSON, err := marshalStringArray(p.ACMEDirectories)
    if err != nil {
        return err
    }

    _, err = db.conn.ExecContext(ctx, `
        INSERT INTO provisioners (name, type, jwk, acme_directories)
        VALUES (?, ?, ?, ?)
    `, p.Name, p.Type, p.JWK, dirsJSON)

    return err
}

func (db *Database) DeleteProvisioner(ctx context.Context, name string) error {
    _, err := db.conn.ExecContext(ctx, `DELETE FROM provisioners WHERE name = ?`, name)
    return err
}

func (db *Database) ListProvisionerStatuses(ctx context.Context) ([]ProvisionerStatus, error) {
    provs, err := db.ListProvisioners(ctx)
    if err != nil {
        return nil, err
    }
    out := make([]ProvisionerStatus, 0, len(provs))
    for _, p := range provs {
        // Minimal implementation: mark as online for now.
        out = append(out, ProvisionerStatus{Name: p.Name, Status: "online"})
    }
    return out, nil
}

// -----------------------------
// Certificates (context-aware)
// -----------------------------

func (db *Database) ListCertificates(ctx context.Context) ([]Certificate, error) {
    rows, err := db.conn.QueryContext(ctx, `
        SELECT id, common_name, dns_names, serial, not_before, not_after,
               certificate_pem, private_key_pem, ca_chain_pem, status, created_at
        FROM certificates
        ORDER BY created_at DESC
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []Certificate

    for rows.Next() {
        var c Certificate
        var dns sql.NullString
        var notBefore sql.NullTime
        var notAfter sql.NullTime
        var created sql.NullTime
        if err := rows.Scan(
            &c.ID, &c.CommonName, &dns, &c.Serial,
            &notBefore, &notAfter,
            &c.CertificatePEM, &c.PrivateKeyPEM, &c.CAChainPEM, &c.Status, &created,
        ); err != nil {
            return nil, err
        }
        if dns.Valid {
            _ = json.Unmarshal([]byte(dns.String), &c.DNSNames)
        }
        if notBefore.Valid {
            c.NotBefore = notBefore.Time
        }
        if notAfter.Valid {
            c.NotAfter = notAfter.Time
        }
        if created.Valid {
            c.CreatedAt = created.Time
        }
        out = append(out, c)
    }

    return out, nil
}

func (db *Database) GetCertificate(ctx context.Context, id string) (*Certificate, error) {
    row := db.conn.QueryRowContext(ctx, `
        SELECT id, common_name, dns_names, serial, not_before, not_after,
               certificate_pem, private_key_pem, ca_chain_pem, status, created_at
        FROM certificates
        WHERE id = ?
    `, id)

    var c Certificate
    var dns sql.NullString
    var notBefore sql.NullTime
    var notAfter sql.NullTime
    var created sql.NullTime

    if err := row.Scan(
        &c.ID, &c.CommonName, &dns, &c.Serial,
        &notBefore, &notAfter,
        &c.CertificatePEM, &c.PrivateKeyPEM, &c.CAChainPEM, &c.Status, &created,
    ); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    if dns.Valid {
        _ = json.Unmarshal([]byte(dns.String), &c.DNSNames)
    }
    if notBefore.Valid {
        c.NotBefore = notBefore.Time
    }
    if notAfter.Valid {
        c.NotAfter = notAfter.Time
    }
    if created.Valid {
        c.CreatedAt = created.Time
    }
    return &c, nil
}

func (db *Database) CreateCertificate(ctx context.Context, c *Certificate) (string, error) {
    if c.ID == "" {
        c.ID = NewID()
    }
    dnsJSON, err := marshalStringArray(c.DNSNames)
    if err != nil {
        return "", err
    }
    _, err = db.conn.ExecContext(ctx, `
        INSERT INTO certificates (id, common_name, dns_names, serial, not_before, not_after,
                                  certificate_pem, private_key_pem, ca_chain_pem, status)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, c.ID, c.CommonName, dnsJSON, c.Serial, c.NotBefore, c.NotAfter,
        c.CertificatePEM, c.PrivateKeyPEM, c.CAChainPEM, c.Status)
    if err != nil {
        return "", err
    }
    return c.ID, nil
}

func (db *Database) RevokeCertificate(ctx context.Context, serial string) error {
    // Minimal approach: mark status revoked for matching serial(s)
    _, err := db.conn.ExecContext(ctx, `UPDATE certificates SET status = ? WHERE serial = ?`, "revoked", serial)
    return err
}

// -----------------------------
// Audit
// -----------------------------

func (db *Database) LogAudit(ctx context.Context, action, user, details, ip string) error {
    _, err := db.conn.ExecContext(ctx, `
        INSERT INTO audit_events (action, user, details, ip)
        VALUES (?, ?, ?, ?)
    `, action, user, details, ip)
    return err
}

// -----------------------------
// CA Settings
// -----------------------------

func (db *Database) GetCASettings(ctx context.Context) (*CASettings, error) {
    row := db.conn.QueryRowContext(ctx, `SELECT provisioner_name, acme_directories, updated_at FROM ca_settings WHERE id = 1`)
    var prov sql.NullString
    var dirs sql.NullString
    var updated sql.NullTime
    if err := row.Scan(&prov, &dirs, &updated); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    out := &CASettings{}
    if prov.Valid {
        out.ProvisionerName = prov.String
    }
    if dirs.Valid {
        _ = json.Unmarshal([]byte(dirs.String), &out.ACMEDirectories)
    }
    if updated.Valid {
        out.UpdatedAt = updated.Time
    }
    return out, nil
}

func (db *Database) UpdateCASettings(ctx context.Context, s *CASettings) error {
    dirsJSON, err := marshalStringArray(s.ACMEDirectories)
    if err != nil {
        return err
    }
    _, err = db.conn.ExecContext(ctx, `UPDATE ca_settings SET provisioner_name = ?, acme_directories = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1`, s.ProvisionerName, dirsJSON)
    return err
}
