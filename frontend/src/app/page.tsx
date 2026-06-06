'use client'

import { useEffect, useState } from 'react'
import { apiClient, CertificateItem, AuditEvent, Provisioner } from '@/src/lib/api'
import { Button } from '@/components/ui/button'
import { toast } from "@/components/ui/use-toast"

export default function DashboardPage() {
  const [caOnline, setCaOnline] = useState<boolean | null>(null)
  const [provisioners, setProvisioners] = useState<Provisioner[]>([])
  const [certs, setCerts] = useState<CertificateItem[]>([])
  const [audit, setAudit] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(true)

  async function load() {
    try {
      setLoading(true)

      // CA health
      try {
        await apiClient.health()
        setCaOnline(true)
      } catch {
        setCaOnline(false)
      }

      // Provisioners
      const prov = await apiClient.listProvisioners()
      setProvisioners(prov.items)

      // Certificates
      const cert = await apiClient.listCertificates()
      setCerts(cert.items)

      // Audit log
      const auditRes = await apiClient.getAuditLog()
      setAudit(auditRes.items)

    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  if (loading) return <div className="p-6">Loading…</div>

  // Derived metrics
  const expired = certs.filter(c => new Date(c.not_after) < new Date()).length
  const revoked = certs.filter(c => c.status === 'revoked').length
  const issued = certs.length

  const recentCerts = [...certs]
    .sort((a, b) => new Date(b.not_before).getTime() - new Date(a.not_before).getTime())
    .slice(0, 5)

  const recentAudit = [...audit]
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    .slice(0, 5)

  return (
    <div className="page page-dashboard space-y-6">

      {/* STATUS CARDS */}
      <section className="cards-row">
        <div className={`card card-status ${caOnline ? 'card-status-ok' : 'card-status-danger'}`}>
          <div className="card-title">CA Status</div>
          <div className="card-value">{caOnline ? 'Online' : 'Offline'}</div>
        </div>

        <div className="card card-status">
          <div className="card-title">Active Provisioners</div>
          <div className="card-value">{provisioners.length}</div>
        </div>

        <div className="card card-status card-status-info">
          <div className="card-title">Certificates Issued</div>
          <div className="card-value">{issued}</div>
        </div>

        <div className="card card-status card-status-danger">
          <div className="card-title">Certificates Expired</div>
          <div className="card-value">{expired}</div>
        </div>
      </section>

      {/* TABLES */}
      <section className="grid-two-columns">

        {/* RECENT CERTIFICATES */}
        <div className="card card-table">
          <div className="card-header">
            <h2 className="card-header-title">Recent Certificates</h2>
          </div>

          <div className="card-body">
            <table className="table w-full">
              <thead>
                <tr>
                  <th>Common Name</th>
                  <th>Serial</th>
                  <th>Status</th>
                  <th>Expires</th>
                </tr>
              </thead>
              <tbody>
                {recentCerts.map(c => (
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
                    <td>{new Date(c.not_after).toLocaleDateString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* RECENT AUDIT EVENTS */}
        <div className="card card-table">
          <div className="card-header">
            <h2 className="card-header-title">Recent Audit Events</h2>
          </div>

          <div className="card-body">
            <table className="table w-full">
              <thead>
                <tr>
                  <th>Event</th>
                  <th>User</th>
                  <th>Timestamp</th>
                </tr>
              </thead>
              <tbody>
                {recentAudit.map(ev => (
                  <tr key={ev.id}>
                    <td>{ev.action}</td>
                    <td>{ev.user}</td>
                    <td>{new Date(ev.timestamp).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

      </section>
    </div>
  )
}
