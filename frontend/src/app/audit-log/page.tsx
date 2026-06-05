'use client'

import { useEffect, useState } from 'react'
import { apiClient, AuditEvent } from '@/src/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'

// ------------------------------------------------------------
// PAGE
// ------------------------------------------------------------

export default function AuditLogPage() {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Filters
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [action, setAction] = useState('')
  const [user, setUser] = useState('')

  // Detail dialog
  const [detailOpen, setDetailOpen] = useState(false)
  const [selected, setSelected] = useState<AuditEvent | null>(null)

  async function load() {
    try {
      setLoading(true)
      const res = await apiClient.getAuditLog({
        from: from || undefined,
        to: to || undefined,
        action: action || undefined,
        user: user || undefined,
      })
      setEvents(res.items)
      setError(null)
    } catch (err) {
      setError('Failed to load audit log')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  if (loading) return <div className="p-6">Loading…</div>
  if (error) return <div className="p-6 text-red-500">{error}</div>

  return (
    <div className="page page-audit-log space-y-6">

      {/* FILTERS */}
      <section className="card">
        <div className="card-header">
          <h2 className="card-header-title">Audit Log Filters</h2>
        </div>

        <div className="card-body grid grid-cols-1 md:grid-cols-4 gap-4">

          <div>
            <Label>From</Label>
            <Input type="datetime-local" value={from} onChange={e => setFrom(e.target.value)} />
          </div>

          <div>
            <Label>To</Label>
            <Input type="datetime-local" value={to} onChange={e => setTo(e.target.value)} />
          </div>

          <div>
            <Label>Action</Label>
            <Input placeholder="certificate_issued" value={action} onChange={e => setAction(e.target.value)} />
          </div>

          <div>
            <Label>User</Label>
            <Input placeholder="system" value={user} onChange={e => setUser(e.target.value)} />
          </div>

          <div className="md:col-span-4">
            <Button onClick={load}>Apply Filters</Button>
          </div>
        </div>
      </section>

      {/* TABLE */}
      <section className="card card-table">
        <div className="card-header">
          <h2 className="card-header-title">Audit Log</h2>
        </div>

        <div className="card-body">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Timestamp</th>
                <th>Action</th>
                <th>User</th>
                <th>Details</th>
                <th>IP</th>
                <th></th>
              </tr>
            </thead>

            <tbody>
              {events.map(ev => (
                <tr key={ev.id}>
                  <td>{new Date(ev.timestamp).toLocaleString()}</td>
                  <td>{ev.action}</td>
                  <td>{ev.user}</td>
                  <td>{ev.details}</td>
                  <td>{ev.ip}</td>
                  <td>
                    <Button
                      size="sm"
                      onClick={() => {
                        setSelected(ev)
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

      {/* DETAIL DIALOG */}
      <AuditDetailDialog
        open={detailOpen}
        event={selected}
        onClose={() => setDetailOpen(false)}
      />
    </div>
  )
}

// ------------------------------------------------------------
// DETAIL DIALOG
// ------------------------------------------------------------

function AuditDetailDialog({ open, event, onClose }: {
  open: boolean
  event: AuditEvent | null
  onClose: () => void
}) {
  if (!event) return null

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Audit Event Details</DialogTitle>
        </DialogHeader>

        <div className="space-y-2">
          <div><strong>ID:</strong> {event.id}</div>
          <div><strong>Timestamp:</strong> {new Date(event.timestamp).toLocaleString()}</div>
          <div><strong>Action:</strong> {event.action}</div>
          <div><strong>User:</strong> {event.user}</div>
          <div><strong>IP:</strong> {event.ip}</div>
          <div><strong>Details:</strong> {event.details}</div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
