'use client';

import { useState } from 'react';

export default function SettingsPage() {
  // CA CONFIG
  const [caUrl, setCaUrl] = useState('https://ca.example.local');
  const [rootFingerprint, setRootFingerprint] = useState(
    '12:AB:34:CD:56:EF:78:90'
  );
  const [caStatus, setCaStatus] = useState<'Online' | 'Offline'>('Online');

  // PROVISIONER CONFIG
  const [provName, setProvName] = useState('web-services');
  const [provPassword, setProvPassword] = useState('');
  const [provType, setProvType] = useState<'ACME' | 'JWK' | 'SSHPOP'>('ACME');

  // ACME DIRECTORIES
  const [directories, setDirectories] = useState<string[]>([
    'https://ca.example.local/acme/web-services/directory',
  ]);

  const addDirectory = () => {
    setDirectories([...directories, '']);
  };

  const updateDirectory = (index: number, value: string) => {
    const updated = [...directories];
    updated[index] = value;
    setDirectories(updated);
  };

  const removeDirectory = (index: number) => {
    setDirectories(directories.filter((_, i) => i !== index));
  };

  return (
    <div className="page page-settings">

      {/* CA CONFIGURATION */}
      <section className="card">
        <div className="card-header">
          <h2 className="card-header-title">CA Configuration</h2>
        </div>

        <div className="card-body">
          <div className="detail-row">
            <label><strong>CA URL:</strong></label>
            <input
              className="input"
              value={caUrl}
              onChange={(e) => setCaUrl(e.target.value)}
            />
          </div>

          <div className="detail-row">
            <label><strong>Root Fingerprint:</strong></label>
            <input
              className="input"
              value={rootFingerprint}
              onChange={(e) => setRootFingerprint(e.target.value)}
            />
          </div>

          <div className="detail-row">
            <label><strong>CA Status:</strong></label>
            <select
              className="input"
              value={caStatus}
              onChange={(e) => setCaStatus(e.target.value as any)}
            >
              <option value="Online">Online</option>
              <option value="Offline">Offline</option>
            </select>
          </div>

          <div className="detail-actions">
            <button className="btn btn-primary">Save CA Configuration</button>
          </div>
        </div>
      </section>

      {/* PROVISIONER CONFIGURATION */}
      <section className="card">
        <div className="card-header">
          <h2 className="card-header-title">Provisioner Configuration</h2>
        </div>

        <div className="card-body">
          <div className="detail-row">
            <label><strong>Provisioner Name:</strong></label>
            <input
              className="input"
              value={provName}
              onChange={(e) => setProvName(e.target.value)}
            />
          </div>

          <div className="detail-row">
            <label><strong>Password:</strong></label>
            <input
              className="input"
              type="password"
              value={provPassword}
              onChange={(e) => setProvPassword(e.target.value)}
            />
          </div>

          <div className="detail-row">
            <label><strong>Type:</strong></label>
            <select
              className="input"
              value={provType}
              onChange={(e) => setProvType(e.target.value as any)}
            >
              <option value="ACME">ACME</option>
              <option value="JWK">JWK</option>
              <option value="SSHPOP">SSHPOP</option>
            </select>
          </div>

          <div className="detail-actions">
            <button className="btn btn-primary">Save Provisioner Settings</button>
          </div>
        </div>
      </section>

      {/* ACME DIRECTORIES */}
      <section className="card">
        <div className="card-header">
          <h2 className="card-header-title">ACME Directories</h2>
        </div>

        <div className="card-body">
          {directories.map((dir, index) => (
            <div className="detail-row" key={index}>
              <input
                className="input"
                value={dir}
                placeholder="https://example.com/acme/directory"
                onChange={(e) => updateDirectory(index, e.target.value)}
              />
              <button
                className="btn btn-danger btn-small"
                onClick={() => removeDirectory(index)}
              >
                Remove
              </button>
            </div>
          ))}

          <button className="btn btn-secondary" onClick={addDirectory}>
            Add Directory
          </button>

          <div className="detail-actions">
            <button className="btn btn-primary">Save ACME Directories</button>
          </div>
        </div>
      </section>
    </div>
  );
}
