// Sdílené typy používané ve frontendu pro komunikaci s backendem.

// ---------------------------------------------------------
// Provisioners
// ---------------------------------------------------------

export interface Provisioner {
  name: string;
  type: string;
  acme_directories?: string[];
}

// ---------------------------------------------------------
// Certificates
// ---------------------------------------------------------

export interface CertificateItem {
  id: string;
  common_name: string;
  dns_names: string[];
  serial: string;
  not_before: string;
  not_after: string;
}

export interface CertificateDetail {
  id: string;
  common_name: string;
  dns_names: string[];
  serial: string;
  not_before: string;
  not_after: string;
  certificate_pem: string;
  ca_bundle_pem: string;
}

export interface IssueCertificateRequest {
  common_name: string;
  dns_names: string[];
  not_after_days?: number;
}

export interface IssueCertificateResponse {
  id: string;
  serial: string;
  common_name: string;
  not_before: string;
  not_after: string;
  certificate_pem: string;
  ca_bundle_pem: string;
}

export interface RevokeRequest {
  serial: string;
  secret?: string;
}

// ---------------------------------------------------------
// Audit log
// ---------------------------------------------------------

export interface AuditEvent {
  id: string;
  timestamp: string;
  action: string;
  user: string;
  details: string;
  ip: string;
}

// ---------------------------------------------------------
// Generic list wrapper
// ---------------------------------------------------------

export interface ListResponse<T> {
  items: T[];
}
