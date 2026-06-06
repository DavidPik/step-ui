'use client'

import { useEffect, useState } from 'react'
import { apiClient, CASettings } from '@/src/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from "@/components/ui/use-toast"

export default function SettingsPage() {
  const [settings, setSettings] = useState<CASettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  // Local editable state
  const [caUrl, setCaUrl] = useState('')
  const [fingerprint, setFingerprint] = useState('')
  const [provisionerName, setProvisionerName] = useState('')
  const [directories, setDirectories] = useState<string[]>([])

  async function load() {
    try {
      setLoading(true)
      const data = await apiClient.getCASettings()
      setSettings(data)

      setCaUrl(data.ca_url)
      setFingerprint(data.root_fingerprint)
      setProvisionerName(data.provisioner_name)
      setDirectories(data.acme_directories || [])

      setError(null)
    } catch (err) {
      setError('Failed to load CA settings')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  async function save() {
    setSaving(true)
    try {
      await apiClient.updateCASettings({
        ca_url: caUrl,
        root_fingerprint: fingerprint,
        provisioner_name: provisionerName,
        acme_directories: directories,
      })
      await load()
      toast({
        title: "Settings saved",
      })
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Failed to save settings",
      })
    } finally {
      setSaving(false)
    }
  }

  function updateDirectory(index: number, value: string) {
    const updated = [...directories]
    updated[index] = value
    setDirectories(updated)
  }

  function addDirectory() {
    setDirectories([...directories, ''])
  }

  function removeDirectory(index: number) {
    setDirectories(directories.filter((_, i) => i !== index))
  }

  if (loading) return <div className="p-6">Loading…</div>
  if (error) return <div className="p-6 text-red-500">{error}</div>

  return (
    <div className="page page-settings space-y-6">

      {/* CA SETTINGS */}
      <section className="card">
        <div className="card-header">
          <h2 className="card-header-title">CA Settings</h2>
        </div>

        <div className="card-body space-y-4">

          <div>
            <Label>CA URL</Label>
            <Input value={caUrl} onChange={e => setCaUrl(e.target.value)} />
          </div>

          <div>
            <Label>Root Fingerprint</Label>
            <Input value={fingerprint} onChange={e => setFingerprint(e.target.value)} />
          </div>

          <div>
            <Label>Provisioner Name</Label>
            <Input value={provisionerName} onChange={e => setProvisionerName(e.target.value)} />
          </div>

          <div className="space-y-2">
            <Label>ACME Directories</Label>

            {directories.map((dir, index) => (
              <div key={index} className="flex gap-2">
                <Input
                  value={dir}
                  placeholder="https://example.com/acme/directory"
                  onChange={e => updateDirectory(index, e.target.value)}
                />
                <Button
                  variant="destructive"
                  onClick={() => removeDirectory(index)}
                >
                  Remove
                </Button>
              </div>
            ))}

            <Button variant="secondary" onClick={addDirectory}>
              Add Directory
            </Button>
          </div>

          <div className="pt-4">
            <Button onClick={save} disabled={saving}>
              {saving ? 'Saving…' : 'Save Settings'}
            </Button>
          </div>
        </div>
      </section>
    </div>
  )
}
