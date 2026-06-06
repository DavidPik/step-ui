//
// Provisioners
//

export interface Provisioner {
  name: string;
  type: string;
  jwk?: string;
  acme_directories: string[];
}

export interface ProvisionerStatus {
  name: string;
  status: "online" | "offline" | "error" | "unknown";
}

export interface ProvisionerStatusList {
  items: ProvisionerStatus[];
}

//
// Certificates
//

export interface CertificateItem {
  id: string;
  common_name: string;
  dns_names: string; // CSV string from backend
  serial: string;
  not_before: string;
  not_after: string;
}

export interface CertificateDetail {
  id: string;
  common_name: string;
  dns_names: string; // CSV string
  serial: string;
  not_before: string;
  not_after: string;
  certificate_pem: string;
  private_key_pem: string;
  ca_chain_pem: string;
}

export interface IssueCertificateRequest {
  common_name: string;
  dns_names: string[];
}

export interface IssueCertificateResponse {
  status: string;
  id: string;
  common_name: string;
  serial: string;
  not_before: string;
  not_after: string;
  certificate: string;
  private_key: string;
  ca_bundle: string;
}

//
// Audit log
//

export interface AuditEvent {
  id: string;
  timestamp: string;
  action: string;
  user: string;
  details: string;
  ip: string;
}

//
// Generic list wrapper
//

export interface ListResponse<T> {
  items: T[];
}
