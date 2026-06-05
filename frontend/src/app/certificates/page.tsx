'use client';

import { useState } from 'react';

type Certificate = {
  cn: string;
  serial: string;
  status: 'Active' | 'Expired' | 'Revoked';
  issued: string;
  expires: string;
  provisioner: string;
};

const CERTIFICATES: Certificate[] = [
  {
    cn: 'www.example.com',
    serial: '1234567890ABCDEF',
    status: 'Active',
    issued: '2024-05-10',
    expires: '2025-05-10',
    provisioner: 'web-services',
  },
  {
    cn: 'api.service.local',
    serial: 'A1B2C3D4E5F6',
    status: 'Active',
    issued: '2024-05-11',
    expires: '2025-05-11',
    provisioner: 'web-services',
  },
  {
    cn: 'test.domain.net',
    serial: '98765432109876',
    status: 'Revoked',
    issued: '2024-05-12',
    expires: '2025-05-12',
    provisioner: 'bootstrap',
  },
  {
    cn: 'old.example.org',
    serial: '11223344556677',
    status: 'Expired',
    issued: '2023-05-10',
    expires: '2024-05-10',
    provisioner: 'bootstrap',
  },
];

export default function CertificatesPage() {
  const [selected, setSelected] = useState<Certificate | null>(null);
  const [statusFilter, setStatusFilter] = useState<'All' | Certificate['status']>('All');
  const [search, setSearch] = useState('');

  const filtered = CERTIFICATES.filter((c) => {
    const matchesStatus = statusFilter === 'All' || c.status === statusFilter;
    const matchesSearch =
      c.cn.toLowerCase().includes(search.toLowerCase()) ||
      c.serial.toLowerCase().includes(search.toLowerCase());
    return matchesStatus && matchesSearch;
  });

  return (
    <div className="page page-certificates">

      {/* ACTIONS + FILTERS */}
      <section className="card">
        <div className="card-body" style={{ display: 'flex', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
          
          {/* ACTION BUTTONS */}
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <button className="btn btn-primary">Issue Certificate</button>
            <button className="btn btn-success">Sign CSR</button>
          </div>

          {/* FILTERS */}
          <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', flexWrap: 'wrap' }}>
            <label>Status:</label>
            <select
              className="input"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as any)}
            >
              <option value="All">All</option>
              <option value="Active">Active</option>
              <option value="Expired">Expired</option>
              <option value="Revoked">Revoked</option>
            </select>

            <input
              className="input"
              placeholder="Search..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />

            <select className="input">
              <option>Show 10</option>
              <option>Show 25</option>
              <option>Show 50</option>
            </select>
          </div>
        </div>
      </section>

      {/* TABLE */}
      <section className="card card-table">
        <div className="card-header">
          <h2 className="card-header-title">Certificates</h2>
        </div>

        <div className="card-body">
          <table className="table">
            <thead>
              <tr>
                <th>Common Name</th>
                <th>Serial</th>
                <th>Status</th>
                <th>Issued</th>
                <th>Expires</th>
                <th>Actions</th>
              </tr>
            </thead>

            <tbody>
              {filtered.map((c) => (
                <tr key={c.serial}>
                  <td>{c.cn}</td>
                  <td>{c.serial}</td>
                  <td
                    className={
                      c.status === 'Active'
                        ? 'badge badge-ok'
                        : c.status === 'Revoked'
                        ? 'badge badge-danger'
                        : 'badge'
                    }
                  >
                    {c.status}
                  </td>
                  <td>{c.issued}</td>
                  <td>{c.expires}</td>
                  <td>
                    <button
                      className="btn btn-small btn-primary"
                      onClick={() => setSelected(c)}
                    >
                      View
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* DETAILS */}
      {selected && (
        <section className="card">
          <div className="card-header">
            <h2 className="card-header-title">Certificate Details</h2>
          </div>

          <div className="card-body">
            <div className="detail-row"><strong>Common Name:</strong> {selected.cn}</div>
            <div className="detail-row"><strong>Serial:</strong> {selected.serial}</div>
            <div className="detail-row"><strong>Status:</strong> {selected.status}</div>
            <div className="detail-row"><strong>Issued:</strong> {selected.issued}</div>
            <div className="detail-row"><strong>Expires:</strong> {selected.expires}</div>
            <div className="detail-row"><strong>Provisioner:</strong> {selected.provisioner}</div>
            <div className="detail-row"><strong>Fingerprint:</strong> 12:AB:34:CD:56:EF:78:90</div>

            <div className="detail-row">
              <strong>Download:</strong>
              <div style={{ display: 'flex', gap: '1rem', marginTop: '0.25rem' }}>
                <a href="#" className="link">Certificate (PEM)</a>
                <a href="#" className="link">Chain (PEM)</a>
              </div>
            </div>

            <div className="detail-actions">
              <button className="btn btn-danger">Revoke Certificate</button>
            </div>
          </div>
        </section>
      )}
    </div>
  );
}
