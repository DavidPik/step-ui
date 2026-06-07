package db

import (
    "context"
    "database/sql"
    "encoding/json"
    "errors"
    "time"

    _ "github.com/mattn/go-sqlite3"
    "github.com/google/uuid"
)

type Database struct {
    conn *sql.DB
}

// Models (kept minimal and compatible with handlers)
type Provisioner struct {
    Name            string    `json:"name"`
    Type            string    `json:"type"`
    JWK             string    `json:"jwk,omitempty"`
    ACMEDirectories []string  `json:"acme_directories,omitempty"`
    CreatedAt       time.Time `json:"created_at,omitempty"`
}

type ProvisionerStatus struct {
    Name   string `json:"name"`
    Status string `json:"status"`
}

type Certificate struct {
    ID             string    `json:"id"`
    CommonName     string    `json:"common_name"`
    DNSNames       []string  `json:"dns_names,omitempty"`
    Serial         string    `json:"serial"`
    NotBefore      time.Time `json:"not_before"`
    NotAfter       time.Time `json:"not_after"`
    CertificatePEM string    `json:"certificate_pem,omitempty"`
    PrivateKeyPEM  string    `json:"private_key_pem,omitempty"`
    CAChainPEM     string    `json:"ca_chain_pem,omitempty"`
    Status         string    `json:"status,omitempty"`
    CreatedAt      time.Time `json:"created_at,omitempty"`
}

type AuditEvent struct {
    ID        int64     `json:"id"`
    Timestamp time.Time `json:"timestamp"`
    Action    string    `json:"action"`
    User      string    `json:"user"`
    Details   string    `json:"details"`
    IP        string    `json:"ip"`
}

type CASettings struct {
    ProvisionerName  string   `json:"provisioner_name,omitempty"`
    ACMEDirectories  []string `json:"acme_directories,omitempty"`
    UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// InitDB opens DB and ensures schema exists.
func InitDB(ctx context.Context, dsn string) (*Database, error) {
    conn, err := sql.Open("sqlite3", dsn)
    if err != nil {
        return nil, err
    }
    // limit connections for sqlite
    conn.SetMaxOpenConns(1)

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
    schema := `
CREATE TABLE IF NOT EXISTS provisioners (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    jwk TEXT,
    acme_directories TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS certificates (
    id TEXT PRIMARY KEY,
    common_name TEXT NOT NULL,
    dns_names TEXT,
    serial TEXT,
    not_before TIMESTAMP,
    not_after TIMESTAMP,
    certificate_pem TEXT,
    private_key_pem TEXT,
    ca_chain_pem TEXT,
    status TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    action TEXT NOT NULL,
    user TEXT NOT NULL,
    details TEXT,
    ip TEXT
);

CREATE TABLE IF NOT EXISTS ca_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    provisioner_name TEXT,
    acme_directories TEXT,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO ca_settings (id, provisioner_name, acme_directories) VALUES (1, '', '[]');
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
