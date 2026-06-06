package db

import (
    "context"
    "database/sql"
    "errors"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type Database struct {
    conn *sql.DB
}

func InitDB(ctx context.Context, dsn string) (*Database, error) {
    db, err := sql.Open("sqlite3", dsn)
    if err != nil {
        return nil, err
    }

    schema := `
CREATE TABLE IF NOT EXISTS provisioners (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    jwk TEXT,
    acme_directories TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS certificates (
    id TEXT PRIMARY KEY,
    common_name TEXT NOT NULL,
    dns_names TEXT NOT NULL,
    serial TEXT NOT NULL,
    not_before TIMESTAMP NOT NULL,
    not_after TIMESTAMP NOT NULL,
    certificate_pem TEXT,
    ca_chain_pem TEXT,
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
`
    _, err = db.ExecContext(ctx, schema)
    if err != nil {
        return nil, err
    }

    return &Database{conn: db}, nil
}

func (db *Database) Close() error {
    return db.conn.Close()
}

// -----------------------------
// Provisioners
// -----------------------------

func (db *Database) ListProvisioners() ([]Provisioner, error) {
    rows, err := db.conn.Query(`
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
        var dirsJSON string

        if err := rows.Scan(&p.Name, &p.Type, &p.JWK, &dirsJSON, &p.CreatedAt); err != nil {
            return nil, err
        }

        dirs, err := JSONToStringArray(dirsJSON)
        if err != nil {
            return nil, err
        }
        p.ACMEDirectories = dirs

        out = append(out, p)
    }

    return out, nil
}

func (db *Database) GetProvisionerByName(name string) (*Provisioner, error) {
    row := db.conn.QueryRow(`
        SELECT name, type, jwk, acme_directories, created_at
        FROM provisioners
        WHERE name = ?
    `, name)

    var p Provisioner
    var dirsJSON string

    err := row.Scan(&p.Name, &p.Type, &p.JWK, &dirsJSON, &p.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    dirs, err := JSONToStringArray(dirsJSON)
    if err != nil {
        return nil, err
    }
    p.ACMEDirectories = dirs

    return &p, nil
}

func (db *Database) CreateProvisioner(p *Provisioner) error {
    dirsJSON, err := StringArrayToJSON(p.ACMEDirectories)
    if err != nil {
        return err
    }

    _, err = db.conn.Exec(`
        INSERT INTO provisioners (name, type, jwk, acme_directories)
        VALUES (?, ?, ?, ?)
    `, p.Name, p.Type, p.JWK, dirsJSON)

    return err
}

func (db *Database) DeleteProvisioner(name string) error {
    _, err := db.conn.Exec(`DELETE FROM provisioners WHERE name = ?`, name)
    return err
}

// -----------------------------
// Certificates
// -----------------------------

func (db *Database) ListCertificates() ([]Certificate, error) {
    rows, err := db.conn.Query(`
        SELECT id, common_name, dns_names, serial, not_before, not_after,
               certificate_pem, ca_chain_pem, created_at
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
        if err := rows.Scan(
            &c.ID, &c.CommonName, &c.DNSNames, &c.Serial,
            &c.NotBefore, &c.NotAfter,
            &c.CertificatePEM, &c.CAChainPEM,
            &c.CreatedAt,
        ); err != nil {
            return nil, err
        }
        out = append(out, c)
    }

    return out, nil
}

func (db *Database) GetCertificateByID(id string) (*Certificate, error) {
    row := db.conn.QueryRow(`
        SELECT id, common_name, dns_names, serial, not_before, not_after,
               certificate_pem, ca_chain_pem, created_at
        FROM certificates
        WHERE id = ?
    `, id)

    var c Certificate

    err := row.Scan(
        &c.ID, &c.CommonName, &c.DNSNames, &c.Serial,
        &c.NotBefore, &c.NotAfter,
        &c.CertificatePEM, &c.CAChainPEM,
        &c.CreatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    return &c, nil
}

func (db *Database) CreateCertificate(c *Certificate) (string, error) {
    _, err := db.conn.Exec(`
        INSERT INTO certificates (id, common_name, dns_names, serial, not_before, not_after,
                                  certificate_pem, ca_chain_pem)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `, c.ID, c.CommonName, c.DNSNames, c.Serial, c.NotBefore, c.NotAfter,
        c.CertificatePEM, c.CAChainPEM)

    if err != nil {
        return "", err
    }

    return c.ID, nil
}

// -----------------------------
// Audit log
// -----------------------------

func (db *Database) LogAuditEvent(e *AuditEvent) error {
    _, err := db.conn.Exec(`
        INSERT INTO audit_events (action, user, details, ip)
        VALUES (?, ?, ?, ?)
    `, e.Action, e.User, e.Details, e.IP)
    return err
}
