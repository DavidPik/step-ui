'use client'

import { useEffect, useState } from 'react'
import { apiClient, Provisioner } from '@/src/lib/api'
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
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select'
import { toast } from "@/components/ui/use-toast"
import { Skeleton } from "@/src/lib/skeleton"

// ------------------------------------------------------------
// PAGE
// ------------------------------------------------------------

export default function ProvisionersPage() {
  const [provisioners, setProvisioners] = useState<Provisioner[]>([])
  const [selected, setSelected] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [selectOpen, setSelectOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  // Selected provisioner for dialogs
  const [activeProvisioner, setActiveProvisioner] = useState<Provisioner | null>(null)

  async function load() {
    try {
      setLoading(true)
      const res = await apiClient.listProvisioners()
      setProvisioners(res.items)
      const sel = await apiClient.getSelectedProvisioner()
      setSelected(sel.name)
      setError(null)
    } catch (err: any) {
      setError('Failed to load provisioners')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  if (loading) return <div className="p-6">Loading…</div>
  if (error) return <div className="p-6 text-red-500">{error}</div>

  const selectedProvisioner = provisioners.find(p => p.name === selected) || null

  return (
    <div className="page page-provisioners space-y-6">

      {/* TABLE */}
      <section className="card card-table">
        <div className="card-header flex justify-between items-center">
          <h2 className="card-header-title">Provisioners</h2>
          <Button onClick={() => setCreateOpen(true)}>+ Create New Provisioner</Button>
        </div>

        <div className="card-body">
          <table className="table w-full">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Active</th>
                <th>Actions</th>
              </tr>
            </thead>

            <tbody>
              {provisioners.map((p) => (
                <tr key={p.name}>
                  <td>{p.name}</td>
                  <td>{p.type}</td>
                  <td>
                    {selected === p.name ? (
                      <span className="badge badge-ok">Active</span>
                    ) : (
                      <span className="badge badge-danger">Inactive</span>
                    )}
                  </td>
                  <td className="space-x-2">
                    <Button
                      size="sm"
                      variant={selected === p.name ? 'default' : 'outline'}
                      onClick={() => {
                        setActiveProvisioner(p)
                        setSelectOpen(true)
                      }}
                    >
                      Select
                    </Button>

                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => {
                        setActiveProvisioner(p)
                        setDeleteOpen(true)
                      }}
                    >
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* DETAILS */}
      {selectedProvisioner && (
        <section className="card">
          <div className="card-header">
            <h2 className="card-header-title">Provisioner Details</h2>
          </div>

          <div className="card-body space-y-2">
            <div><strong>Name:</strong> {selectedProvisioner.name}</div>
            <div><strong>Type:</strong> {selectedProvisioner.type}</div>
            <div><strong>Status:</strong> Active</div>
          </div>
        </section>
      )}

      {/* DIALOGS */}
      <CreateProvisionerDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={load}
      />

      <SelectProvisionerDialog
        open={selectOpen}
        provisioner={activeProvisioner}
        onClose={() => setSelectOpen(false)}
        onSelected={load}
      />

      <DeleteProvisionerDialog
        open={deleteOpen}
        provisioner={activeProvisioner}
        onClose={() => setDeleteOpen(false)}
        onDeleted={load}
      />
    </div>
  )
}

// ------------------------------------------------------------
// CREATE PROVISIONER DIALOG
// ------------------------------------------------------------

function CreateProvisionerDialog({ open, onClose, onCreated }: {
  open: boolean
  onClose: () => void
  onCreated: () => void
}) {
  const [name, setName] = useState('')
  const [type, setType] = useState('JWK')
  const [secret, setSecret] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleCreate() {
    setLoading(true)
    try {
      await apiClient.createProvisioner({
        name,
        type,
        secret: type === 'JWK' ? secret : undefined,
      })
      onCreated()
      toast({
        title: "Provisioner created successfully",
      })
      onClose()
    } catch (err) {
        toast({
          variant: "destructive",
          title: "Failed to create provisioner",
        })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Provisioner</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Name</Label>
            <Input value={name} onChange={e => setName(e.target.value)} />
          </div>

          <div>
            <Label>Type</Label>
            <Select value={type} onValueChange={setType}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="JWK">JWK</SelectItem>
                <SelectItem value="ACME">ACME</SelectItem>
                <SelectItem value="SSHPOP">SSHPOP</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {type === 'JWK' && (
            <div>
              <Label>Secret</Label>
              <Input type="password" value={secret} onChange={e => setSecret(e.target.value)} />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleCreate} disabled={loading}>
            {loading ? 'Creating…' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ------------------------------------------------------------
// SELECT PROVISIONER DIALOG
// ------------------------------------------------------------

function SelectProvisionerDialog({ open, provisioner, onClose, onSelected }: {
  open: boolean
  provisioner: Provisioner | null
  onClose: () => void
  onSelected: () => void
}) {
  const [secret, setSecret] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSelect() {
    if (!provisioner) return
    setLoading(true)
    try {
      await apiClient.selectProvisioner(provisioner.name, secret)
      onSelected()
      toast({
        title: "Provisioner selected",
      })
      onClose()
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Failed to select provisioner",
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Select Provisioner</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Provisioner</Label>
            <Input value={provisioner?.name || ''} disabled />
          </div>

          <div>
            <Label>Secret</Label>
            <Input type="password" value={secret} onChange={e => setSecret(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSelect} disabled={loading}>
            {loading ? 'Selecting…' : 'Select'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ------------------------------------------------------------
// DELETE PROVISIONER DIALOG
// ------------------------------------------------------------

function DeleteProvisionerDialog({ open, provisioner, onClose, onDeleted }: {
  open: boolean
  provisioner: Provisioner | null
  onClose: () => void
  onDeleted: () => void
}) {
  const [loading, setLoading] = useState(false)

  async function handleDelete() {
    if (!provisioner) return
    setLoading(true)
    try {
      await apiClient.deleteProvisioner(provisioner.name)
      onDeleted()
      toast({
        title: "Provisioner deleted",
      })
      onClose()
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Failed to delete provisioner",
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Provisioner</DialogTitle>
        </DialogHeader>

        <p>Are you sure you want to delete <strong>{provisioner?.name}</strong>?</p>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button variant="destructive" onClick={handleDelete} disabled={loading}>
            {loading ? 'Deleting…' : 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
