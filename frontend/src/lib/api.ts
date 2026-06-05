import axios from 'axios'

// Jediný API klient
const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

// -----------------------------
// Typy odpovídající backendu
// -----------------------------

export interface CertificateItem {
  id: string
  common_name: string
  dns_names: string
  serial: string
  not_before: string
  not_after: string
}

export interface CertificateDetail {
  id: string
  common_name: string
  dns_names: string
  serial: string
  not_before: string
  not_after: string
  certificate_pem: string
  private_key_pem: string
  ca_chain_pem: string
}

export interface IssueCertificateRequest {
  common_name: string
  dns_names: string[]
}

export interface IssueCertificateResponse {
  status: string
  id: string
  common_name: string
  serial: string
  not_before: string
  not_after: string
  certificate: string
  private_key: string
  ca_bundle: string
}

export interface RevokeRequest {
  serial: string
}

export interface CASettings {
  ca_url: string
  root_fingerprint: string
  provisioner_name: string
  acme_directories: string[]
}

export interface Provisioner {
  name: string
  type: string
}

export interface AuditEvent {
  id: string
  timestamp: string
  action: string
  user: string
  details: string
  ip: string
}

// -----------------------------
// API volání
// -----------------------------

export const apiClient = {
  //
  // CA SETTINGS
  //
  getCASettings: async (): Promise<CASettings> => {
    const res = await api.get('/settings')
    return res.data
  },

  updateCASettings: async (data: CASettings) => {
    const res = await api.put('/settings', data)
    return res.data
  },

  //
  // PROVISIONERS
  //
  listProvisioners: async (): Promise<{ items: Provisioner[] }> => {
    const res = await api.get('/provisioners')
    return res.data
  },

  getSelectedProvisioner: async (): Promise<{ name: string }> => {
    const res = await api.get('/provisioners/selected')
    return res.data
  },

  selectProvisioner: async (name: string, secret: string) => {
    const res = await api.post('/provisioners/select', { name, secret })
    return res.data
  },

  createProvisioner: async (data: {
    name: string
    type: string
    secret?: string
  }) => {
    const res = await api.post('/provisioners', data)
    return res.data
  },

  deleteProvisioner: async (name: string) => {
    const res = await api.delete(`/provisioners/${name}`)
    return res.data
  },

  //
  // CERTIFICATES
  //
  listCertificates: async (): Promise<{ items: CertificateItem[] }> => {
    const res = await api.get('/certificates')
    return res.data
  },

  getCertificate: async (id: string): Promise<CertificateDetail> => {
    const res = await api.get(`/certificates/${id}`)
    return res.data
  },

  issueCertificate: async (
    data: IssueCertificateRequest
  ): Promise<IssueCertificateResponse> => {
    const res = await api.post('/certificates/issue', data)
    return res.data
  },

  revokeCertificate: async (serial: string) => {
    const res = await api.post('/certificates/revoke', { serial })
    return res.data
  },

  downloadCertificatePackage: (id: string) => {
    // Vrací URL pro <a href>
    return `/api/certificates/${id}/download`
  },

  //
  // AUDIT LOG
  //
  getAuditLog: async (params?: {
    from?: string
    to?: string
    action?: string
    user?: string
  }): Promise<{ items: AuditEvent[] }> => {
    const res = await api.get('/audit', { params })
    return res.data
  },
}

export default apiClient
