'use client';

import { useEffect, useState } from 'react';
import { apiClient } from '@/lib/api';
import { CertificateItem, CertificateDetail } from '@/lib/types';
import { Button } from '@/components/ui/button';
import { Dialog, DialogHeader, DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { downloadFile } from '@/lib/utils';
import { useActiveProvisioner } from '@/lib/activeProvisioner';

export default function CertificatesPage() {
  const [certs, setCerts] = useState<CertificateItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<'All' | 'Active' | 'Expired'>('All');

  const [issueOpen, setIssueOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [revokeOpen, setRevokeOpen] = useState(false);

  const [selectedCert, setSelectedCert] = useState<CertificateDetail | null>(null);

  const { activeProvisioner, activeProvisionerStatus } = useActiveProvisioner();

  async function load() {
    try {
      setLoading(true);
      const res = await apiClient.listCertificates();
      setCerts(res.items);
      setError(null);
    } catch (err) {
      setError('Failed to load certificates');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  if (loading) return <div className="p-6">Loading…</div>;
  if (error) return <div className="p-6 text-red-500">{error}</div>;

  const filtered = certs.filter((c) => {
    const isExpired = new Date(c.not_after) < new Date();

    const matchesStatus =
      statusFilter === 'All' ||
      (statusFilter === 'Expired' && isExpired) ||
      (statusFilter === 'Active' && !isExpired);

    const matchesSearch =
      c.common_name.toLowerCase().includes(search.toLowerCase()) ||
      c.serial.toLowerCase().includes(search.toLowerCase());

    return matchesStatus && matchesSearch;
  });

  return (
    <div className="page page-certificates space-y-6">

      {/* ACTIONS + FILTERS */}
      <section className="card">
        <div className="card-body flex flex-wrap justify-between gap-4">

          <div className="flex gap-2">
            <Button
              onClick={() => setIssueOpen(true)}
              disabled={!activeProvisioner || activeProvisionerStatus !== 'online'}
            >
              Issue Certificate
            </Button>
          </div>

          <div className="flex gap-2 items-center flex-wrap">
            <label>Status:</label>
            <select
              className="input"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as any)}
            >
              <option value="All">All</option>
              <option value="Active">Active</option>
              <option value="Expired">Expired</option>
            </select>

            <Input
              placeholder="Search CN or Serial…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
        </div>
      </section>

      {/* TABLE */}
      <section className="card card-table">
        <div className="card-header">
          <h2 className="card-header-title">Certificates</h2>
        </div>

        <div className="card-body">
          <table className="table w-full">
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
              {filtered.map((c) => {
                const isExpired = new Date(c.not_after) < new Date();

                return (
                  <tr key={c.serial}>
                    <td>{c.common_name}</td>
                    <td>{c.serial}</td>
                    <td>
                      {isExpired ? (
                        <span className="badge">Expired</span>
                      ) : (
                        <span className="badge badge-ok">Active</span>
                      )}
                    </td>
                    <td>{new Date(c.not_before).toLocaleDateString()}</td>
                    <td>{new Date(c.not_after).toLocaleDateString()}</td>
                    <td>
                      <Button
                        onClick={async () => {
                          const detail = await apiClient.getCertificate(c.id);
                          setSelectedCert(detail);
                          setDetailOpen(true);
                        }}
                      >
                        View
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>

      <IssueCertificateDialog
        open={issueOpen}
        onClose={() => setIssueOpen(false)}
        onIssued={load}
        activeProvisioner={activeProvisioner}
        activeProvisionerStatus={activeProvisionerStatus}
      />

      <CertificateDetailDialog
        open={detailOpen}
        cert={selectedCert}
        onClose={() => setDetailOpen(false)}
        onRevoke={() => {
          setDetailOpen(false);
          setRevokeOpen(true);
        }}
      />

      <RevokeCertificateDialog
        open={revokeOpen}
        cert={selectedCert}
        onClose={() => setRevokeOpen(false)}
        onRevoked={load}
      />
    </div>
  );
}

// ------------------------------------------------------------
// ISSUE CERTIFICATE DIALOG
// ------------------------------------------------------------

function IssueCertificateDialog({
  open,
  onClose,
  onIssued,
  activeProvisioner,
  activeProvisionerStatus,
}: {
  open: boolean;
  onClose: () => void;
  onIssued: () => void;
  activeProvisioner: string | null;
  activeProvisionerStatus: 'online' | 'offline' | 'error' | 'unknown';
}) {
  const [cn, setCn] = useState("");
  const [dns, setDns] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleIssue() {
    setLoading(true);
    try {
      await apiClient.issueCertificate({
        common_name: cn,
        dns_names: dns.split(",").map((s) => s.trim()).filter(Boolean),
      });
      onIssued();
      onClose();
      setCn("");
      setDns("");
    } catch (err) {
      console.error("Failed to issue certificate", err);
    } finally {
      setLoading(false);
    }
  }

  const disabled = !activeProvisioner || activeProvisionerStatus !== 'online';

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-lg">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Issue Certificate</h3>
        </DialogHeader>

        <div className="space-y-4">
          {activeProvisioner ? (
            activeProvisionerStatus !== 'online' ? (
              <div className="p-3 bg-yellow-50 border border-yellow-200 rounded text-sm">
                ⚠️ Active provisioner <strong>{activeProvisioner}</strong> is <strong>{activeProvisionerStatus}</strong>. Issuing certificates is disabled until the provisioner is online.
              </div>
            ) : null
          ) : (
            <div className="p-3 bg-yellow-50 border border-yellow-200 rounded text-sm">
              ⚠️ No active provisioner selected. Select a provisioner in the Provisioners page before issuing certificates.
            </div>
          )}

          <div>
            <Label>Common Name</Label>
            <Input value={cn} onChange={(e) => setCn(e.target.value)} />
          </div>

          <div>
            <Label>DNS Names (comma separated)</Label>
            <Input value={dns} onChange={(e) => setDns(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button onClick={onClose}>Cancel</Button>
          <Button onClick={handleIssue} disabled={loading || disabled}>
            {loading ? "Issuing…" : "Issue"}
          </Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}

// ------------------------------------------------------------
// CERTIFICATE DETAIL DIALOG
// ------------------------------------------------------------

function CertificateDetailDialog({
  open,
  cert,
  onClose,
  onRevoke,
}: {
  open: boolean;
  cert: CertificateDetail | null;
  onClose: () => void;
  onRevoke: () => void;
}) {
  if (!cert) return null;

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-2xl">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Certificate Details</h3>
        </DialogHeader>

        <div className="space-y-2">
          <div><strong>Common Name:</strong> {cert.common_name}</div>
          <div><strong>Serial:</strong> {cert.serial}</div>
          <div><strong>Issued:</strong> {new Date(cert.not_before).toLocaleString()}</div>
          <div><strong>Expires:</strong> {new Date(cert.not_after).toLocaleString()}</div>

          <div>
            <strong>Download:</strong>
            <div className="flex gap-4 mt-1">
              <Button
                onClick={() =>
                  downloadFile(
                    cert.certificate_pem,
                    `${cert.common_name}.crt`,
                    "application/x-pem-file"
                  )
                }
              >
                Certificate (PEM)
              </Button>

              <Button
                onClick={() =>
                  downloadFile(
                    cert.ca_chain_pem,
                    `${cert.common_name}-chain.crt`,
                    "application/x-pem-file"
                  )
                }
              >
                Chain (PEM)
              </Button>

              <Button
                onClick={() => {
                  window.location.href =
                    apiClient.downloadCertificatePackage(cert.id);
                }}
              >
                ZIP Package
              </Button>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button onClick={onRevoke}>Revoke</Button>
          <Button onClick={onClose}>Close</Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}

// ------------------------------------------------------------
// REVOKE CERTIFICATE DIALOG
// ------------------------------------------------------------

function RevokeCertificateDialog({
  open,
  cert,
  onClose,
  onRevoked,
}: {
  open: boolean;
  cert: CertificateDetail | null;
  onClose: () => void;
  onRevoked: () => void;
}) {
  const [loading, setLoading] = useState(false);

  if (!cert) return null;

  async function handleRevoke() {
    setLoading(true);
    try {
      await apiClient.revokeCertificate(cert.serial);
      onRevoked();
      onClose();
    } catch (err) {
      console.error("Failed to revoke certificate", err);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-lg">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Revoke Certificate</h3>
        </DialogHeader>

        <p>
          Are you sure you want to revoke certificate{" "}
          <strong>{cert.common_name}</strong>?
        </p>

        <DialogFooter>
          <Button onClick={onClose}>Cancel</Button>
          <Button onClick={handleRevoke} disabled={loading}>
            {loading ? "Revoking…" : "Revoke"}
          </Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}
