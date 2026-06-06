'use client'

import { useEffect, useState } from 'react'
import { apiClient, CertificateItem, CertificateDetail } from '@/src/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { downloadFile } from '@/src/lib/utils'
import { toast } from "@/components/ui/use-toast"
import { Skeleton } from "@/src/lib/skeleton"

// ------------------------------------------------------------
// PAGE
// ------------------------------------------------------------

export default function CertificatesPage() {
  const [certs, setCerts] = useState<CertificateItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'All' | 'Active' | 'Expired' | 'Revoked'>('All')

  // Dialog states
  const [issueOpen, setIssueOpen] = useState(false)
  const [csrOpen, setCsrOpen] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [revokeOpen, setRevokeOpen] = useState(false)

  const [selectedCert, setSelectedCert] = useState<CertificateDetail | null>(null)

  async function load() {
    try {
      setLoading(true)
      const res = await apiClient.listCertificates()
      setCerts(res.items)
      setError(null)
    } catch (err) {
      setError('Failed to load certificates')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  if (loading) return <div className="p-6">Loading…</div>
  if (error) return <div className="p-6 text-red-500">{error}</div>

  const filtered = certs.filter((c) => {
    const matchesStatus =
      statusFilter === 'All' ||
      (statusFilter === 'Expired' && new Date(c.not_after) < new Date()) ||
      (statusFilter === 'Revoked' && c.status === 'revoked') ||
      (statusFilter === 'Active' && new Date(c.not_after) >= new Date() && c.status !== 'revoked')

    const matchesSearch =
      c.common_name.toLowerCase().includes(search.toLowerCase()) ||
      c.serial.toLowerCase().includes(search.toLowerCase())

    return matchesStatus && matchesSearch
  })

  return (
    <div className="page page-certificates space-y-6">

      {/* ACTIONS + FILTERS */}
      <section className="card">
        <div className="card-body flex flex-wrap justify-between gap-4">

          {/* ACTION BUTTONS */}
          <div className="flex gap-2">
            <Button onClick={() => setIssueOpen(true)}>Issue Certificate</Button>
            <Button variant="secondary" onClick={() => setCsrOpen(true)}>Sign CSR</Button>
          </div>

          {/* FILTERS */}
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
              <option value="Revoked">Revoked</option>
            </select>

            <Input
              placeholder="Search CN or Serial…"
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
              {filtered.map((c) => (
                <tr key={c.serial}>
                  <td>{c.common_name}</td>
                  <td>{c.serial}</td>
                  <td>
                    {c.status === 'revoked' ? (
                      <span className="badge badge-danger">Revoked</span>
                    ) : new Date(c.not_after) < new Date() ? (
                      <span className="badge">Expired</span>
                    ) : (
                      <span className="badge badge-ok">Active</span>
                    )}
                  </td>
                  <td>{new Date(c.not_before).toLocaleDateString()}</td>
                  <td>{new Date(c.not_after).toLocaleDateString()}</td>
                  <td>
                    <Button
                      size="sm"
                      onClick={async () => {
                        const detail = await apiClient.getCertificate(c.id)
                        setSelectedCert(detail)
                        setDetailOpen(true)
                      }}
                    >
                      View
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* DIALOGS */}
      <IssueCertificateDialog
        open={issueOpen}
        onClose={() => setIssueOpen(false)}
        onIssued={load}
      />

      <SignCSRDialog
        open={csrOpen}
        onClose={() => setCsrOpen(false)}
        onSigned={load}
      />

      <CertificateDetailDialog
        open={detailOpen}
        cert={selectedCert}
        onClose={() => setDetailOpen(false)}
        onRevoke={() => {
          setDetailOpen(false)
          setRevokeOpen(true)
        }}
      />

      <RevokeCertificateDialog
        open={revokeOpen}
        cert={selectedCert}
        onClose={() => setRevokeOpen(false)}
        onRevoked={load}
      />
    </div>
  )
}

// ------------------------------------------------------------
// ISSUE CERTIFICATE DIALOG
// ------------------------------------------------------------

function IssueCertificateDialog({ open, onClose, onIssued }: {
  open: boolean
  onClose: () => void
  onIssued: () => void
}) {
  const [cn, setCn] = useState('')
  const [dns, setDns] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleIssue() {
    setLoading(true)
    try {
      await apiClient.issueCertificate({
        common_name: cn,
        dns_names: dns.split(',').map(s => s.trim()).filter(Boolean),
      })
      onIssued()
      toast({
        title: "Certificate issued successfully",
      })
      onClose()
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Failed to issue certificate",
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader><DialogTitle>Issue Certificate</DialogTitle></DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Common Name</Label>
            <Input value={cn} onChange={e => setCn(e.target.value)} />
          </div>

          <div>
            <Label>DNS Names (comma separated)</Label>
            <Input value={dns} onChange={e => setDns(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleIssue} disabled={loading}>
            {loading ? 'Issuing…' : 'Issue'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ------------------------------------------------------------
// SIGN CSR DIALOG
// ------------------------------------------------------------

function SignCSRDialog({ open, onClose, onSigned }: {
  open: boolean
  onClose: () => void
  onSigned: () => void
}) {
  const [csr, setCsr] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSign() {
    setLoading(true)
    try {
      await apiClient.signCSR({
        csr_pem: csr,
        not_after_days: 365,
      })
      onSigned()
      toast({
        title: "CSR signed successfully",
      })
      onClose()
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Failed to sign CSR",
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader><DialogTitle>Sign CSR</DialogTitle></DialogHeader>

        <div>
          <Label>CSR (PEM)</Label>
          <Textarea rows={8} value={csr} onChange={e => setCsr(e.target.value)} />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSign} disabled={loading}>
            {loading ? 'Signing…' : 'Sign'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ------------------------------------------------------------
// CERTIFICATE DETAIL DIALOG
// ------------------------------------------------------------

function CertificateDetailDialog({ open, cert, onClose, onRevoke }: {
  open: boolean
  cert: CertificateDetail | null
  onClose: () => void
  onRevoke: () => void
}) {
  if (!cert) return null

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader><DialogTitle>Certificate Details</DialogTitle></DialogHeader>

        <div className="space-y-2">
          <div><strong>Common Name:</strong> {cert.common_name}</div>
          <div><strong>Serial:</strong> {cert.serial}</div>
          <div><strong>Issued:</strong> {new Date(cert.not_before).toLocaleString()}</div>
          <div><strong>Expires:</strong> {new Date(cert.not_after).toLocaleString()}</div>

          <div>
            <strong>Download:</strong>
            <div className="flex gap-4 mt-1">
              <Button
                variant="outline"
                onClick={() => downloadFile(cert.certificate_pem, `${cert.common_name}.crt`, 'application/x-pem-file')}
              >
                Certificate (PEM)
              </Button>

              <Button
                variant="outline"
                onClick={() => downloadFile(cert.ca_chain_pem, `${cert.common_name}-chain.crt`, 'application/x-pem-file')}
              >
                Chain (PEM)
              </Button>

              <Button
                variant="outline"
                onClick={() => {
                  window.location.href = apiClient.downloadCertificatePackage(cert.id)
                }}
              >
                ZIP Package
              </Button>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="destructive" onClick={onRevoke}>Revoke</Button>
          <Button variant="outline" onClick={onClose}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ------------------------------------------------------------
// REVOKE CERTIFICATE DIALOG
// ------------------------------------------------------------

function RevokeCertificateDialog({ open, cert, onClose, onRevoked }: {
  open: boolean
  cert: CertificateDetail | null
  onClose: () => void
  onRevoked: () => void
}) {
  const [loading, setLoading] = useState(false)

  if (!cert) return null

  async function handleRevoke() {
    setLoading(true)
    try {
      await apiClient.revokeCertificate(cert.serial)
      onRevoked()
      toast({
        title: "Certificate revoked",
      })
      onClose()
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Failed to revoke certificate",
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader><DialogTitle>Revoke Certificate</DialogTitle></DialogHeader>

        <p>
          Are you sure you want to revoke certificate <strong>{cert.common_name}</strong>?
        </p>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button variant="destructive" onClick={handleRevoke} disabled={loading}>
            {loading ? 'Revoking…' : 'Revoke'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
