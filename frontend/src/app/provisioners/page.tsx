'use client';

import { useState } from 'react';

type Provisioner = {
  name: string;
  type: string;
  status: 'Active' | 'Inactive';
  created: string;
};

const PROVISIONERS: Provisioner[] = [
  { name: 'bootstrap', type: 'JWK', status: 'Active', created: '2024-05-10' },
  { name: 'web-services', type: 'ACME', status: 'Active', created: '2024-05-11' },
  { name: 'net-clients', type: 'JWK', status: 'Inactive', created: '2024-05-12' },
  { name: 'ssh-users', type: 'SSHPOP', status: 'Active', created: '2024-05-13' },
];

export default function ProvisionersPage() {
  const [selected, setSelected] = useState<Provisioner | null>(
    PROVISIONERS[1] // default: web-services
  );

  return (
    <div className="page page-provisioners">

      {/* TABLE */}
      <section className="card card-table">
        <div className="card-header">
          <h2 className="card-header-title">Provisioners</h2>
        </div>

        <div className="card-body">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Status</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>

            <tbody>
              {PROVISIONERS.map((p) => (
                <tr key={p.name}>
                  <td>{p.name}</td>
                  <td>{p.type}</td>
                  <td
                    className={
                      p.status === 'Active'
                        ? 'badge badge-ok'
                        : 'badge badge-danger'
                    }
                  >
                    {p.status}
                  </td>
                  <td>{p.created}</td>
                  <td>
                    <button
                      className={
                        'btn btn-small ' +
                        (selected?.name === p.name ? 'btn-primary' : '')
                      }
                      onClick={() => setSelected(p)}
                    >
                      Select
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div style={{ marginTop: '1rem' }}>
            <button className="btn btn-primary">+ Create New Provisioner</button>
          </div>
        </div>
      </section>

      {/* DETAILS */}
      {selected && (
        <section className="card">
          <div className="card-header">
            <h2 className="card-header-title">Provisioner Details</h2>
          </div>

          <div className="card-body">
            <div className="detail-row">
              <strong>Name:</strong> {selected.name}
            </div>
            <div className="detail-row">
              <strong>Type:</strong> {selected.type}
            </div>
            <div className="detail-row">
              <strong>Status:</strong> {selected.status}
            </div>
            <div className="detail-row">
              <strong>Created:</strong> {selected.created}
            </div>

            {/* Mocked values for now */}
            <div className="detail-row">
              <strong>CA URL:</strong> https://ca.example.local
            </div>
            <div className="detail-row">
              <strong>Fingerprint:</strong> 12:AB:34:CD:56:EF:78:90
            </div>
            <div className="detail-row">
              <strong>Certificates Issued:</strong> 42
            </div>

            <div className="detail-actions">
              <button className="btn btn-secondary">Edit Provisioner</button>
              <button className="btn btn-danger">Delete Provisioner</button>
            </div>
          </div>
        </section>
      )}
    </div>
  );
}
