'use client';

import { useEffect, useState } from 'react';
import { apiClient } from '@/lib/api';
import { Provisioner } from '@/lib/types';
import { Button } from '@/components/ui/button';
import { Dialog, DialogHeader, DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/lib/skeleton';

// ------------------------------------------------------------
// PAGE
// ------------------------------------------------------------

export default function ProvisionersPage() {
  const [provisioners, setProvisioners] = useState<Provisioner[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false);
  const [selectOpen, setSelectOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  // Selected provisioner for dialogs
  const [activeProvisioner, setActiveProvisioner] = useState<Provisioner | null>(null);

  async function load() {
    try {
      setLoading(true);
      const res = await apiClient.listProvisioners();
      setProvisioners(res.items);

      // Backend neposílá aktivní provisioner → držíme ho lokálně
      if (!selected && res.items.length > 0) {
        setSelected(res.items[0].name);
      }

      setError(null);
    } catch (err) {
      console.error('Failed to load provisioners', err);
      setError('Failed to load provisioners');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  if (loading) return <div className="p-6">Loading…</div>;
  if (error) return <div className="p-6 text-red-500">{error}</div>;

  const selectedProvisioner =
    provisioners.find((p) => p.name === selected) || null;

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
                      onClick={() => {
                        setActiveProvisioner(p);
                        setSelectOpen(true);
                      }}
                    >
                      Select
                    </Button>

                    <Button
                      onClick={() => {
                        setActiveProvisioner(p);
                        setDeleteOpen(true);
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
  );
}

// ------------------------------------------------------------
// CREATE PROVISIONER DIALOG
// ------------------------------------------------------------

function CreateProvisionerDialog({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}) {
  const [name, setName] = useState("");
  const [type, setType] = useState("JWK");
  const [secret, setSecret] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleCreate() {
    setLoading(true);
    try {
      await apiClient.createProvisioner({
        name,
        type,
        secret: type === "JWK" ? secret : undefined,
      });
      onCreated();
      onClose();
    } catch (err) {
      console.error("Failed to create provisioner", err);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-lg">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Create Provisioner</h3>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div>
            <Label>Type</Label>
            <select
              className="input w-full border rounded px-3 py-2"
              value={type}
              onChange={(e) => setType(e.target.value)}
            >
              <option value="JWK">JWK</option>
              <option value="ACME">ACME</option>
            </select>
          </div>

          {type === "JWK" && (
            <div>
              <Label>Secret</Label>
              <Input
                type="password"
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
              />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button onClick={onClose}>Cancel</Button>
          <Button onClick={handleCreate} disabled={loading}>
            {loading ? "Creating…" : "Create"}
          </Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}

// ------------------------------------------------------------
// SELECT PROVISIONER DIALOG
// ------------------------------------------------------------

function SelectProvisionerDialog({
  open,
  provisioner,
  onClose,
  onSelected,
}: {
  open: boolean;
  provisioner: Provisioner | null;
  onClose: () => void;
  onSelected: () => void;
}) {
  const [secret, setSecret] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSelect() {
    if (!provisioner) return;
    setLoading(true);
    try {
      await apiClient.selectProvisioner(provisioner.name, secret);
      onSelected();
      onClose();
    } catch (err) {
      console.error("Failed to select provisioner", err);
    } finally {
      setLoading(false);
      setSecret("");
    }
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-lg">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Select Provisioner</h3>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Provisioner</Label>
            <Input value={provisioner?.name || ""} disabled />
          </div>

          <div>
            <Label>Secret</Label>
            <Input
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
            />
          </div>
        </div>

        <DialogFooter>
          <Button onClick={onClose}>Cancel</Button>
          <Button onClick={handleSelect} disabled={loading}>
            {loading ? "Selecting…" : "Select"}
          </Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}

// ------------------------------------------------------------
// DELETE PROVISIONER DIALOG
// ------------------------------------------------------------

function DeleteProvisionerDialog({
  open,
  provisioner,
  onClose,
  onDeleted,
}: {
  open: boolean;
  provisioner: Provisioner | null;
  onClose: () => void;
  onDeleted: () => void;
}) {
  const [secret, setSecret] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleDelete() {
    if (!provisioner) return;
    setLoading(true);
    try {
      await apiClient.deleteProvisioner(provisioner.name, secret || undefined);
      onDeleted();
      onClose();
    } catch (err) {
      console.error("Failed to delete provisioner", err);
    } finally {
      setLoading(false);
      setSecret("");
    }
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-lg">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Delete Provisioner</h3>
        </DialogHeader>

        <p>
          Are you sure you want to delete{" "}
          <strong>{provisioner?.name}</strong>?
        </p>

        <div className="space-y-4 mt-4">
          <div>
            <Label>Secret</Label>
            <Input
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              placeholder="Enter provisioner secret to confirm"
            />
          </div>
        </div>

        <DialogFooter>
          <Button onClick={onClose}>Cancel</Button>
          <Button onClick={handleDelete} disabled={loading}>
            {loading ? "Deleting…" : "Delete"}
          </Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}
