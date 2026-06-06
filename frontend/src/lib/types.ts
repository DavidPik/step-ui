// frontend/src/lib/types.ts
// Sdílené typy používané ve frontendu pro komunikaci s backendem.

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
  acme_directories?: string[]
  is_active?: boolean
}

export interface AuditEvent {
  id: string
  timestamp: string
  action: string
  user: string
  details: string
  ip: string
}

// API response wrappers
export interface ListResponse<T> {
  items: T[]
}

export interface SelectedProvisionerResponse {
  name: string
}
