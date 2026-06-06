import axios from "axios";

// ---------------------------------------------------------
// Axios instance
// ---------------------------------------------------------

const api = axios.create({
  baseURL: "/api",
  headers: {
    "Content-Type": "application/json",
  },
});

// ---------------------------------------------------------
// Types (sjednocené s backendem)
// ---------------------------------------------------------

export interface Provisioner {
  name: string;
  type: string;
  acme_directories?: string[];
}

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

export interface AuditEvent {
  id: string;
  timestamp: string;
  action: string;
  user: string;
  details: string;
  ip: string;
}

// ---------------------------------------------------------
// API Client (kompatibilní s backendem)
// ---------------------------------------------------------

export const apiClient = {
  // -----------------------------------------------------
  // PROVISIONERS
  // -----------------------------------------------------

  listProvisioners: async (): Promise<{ items: Provisioner[] }> => {
    const res = await api.get("/provisioners");
    return res.data;
  },

  createProvisioner: async (data: {
    name: string;
    type: string;
    secret?: string;
    acme_directories?: string[];
  }) => {
    const res = await api.post("/provisioners", data);
    return res.data;
  },

  deleteProvisioner: async (name: string, secret?: string) => {
    const payload = secret ? { secret } : {};
    const res = await api.delete(`/provisioners/${encodeURIComponent(name)}`, {
      data: payload,
    });
    return res.data;
  },

  selectProvisioner: async (name: string, secret?: string) => {
    const payload = secret ? { secret } : {};
    const res = await api.post(
      `/provisioners/${encodeURIComponent(name)}/select`,
      payload
    );
    return res.data;
  },

  // -----------------------------------------------------
  // CERTIFICATES
  // -----------------------------------------------------

  listCertificates: async (): Promise<{ items: CertificateItem[] }> => {
    const res = await api.get("/certificates");
    return res.data;
  },

  issueCertificate: async (
    data: IssueCertificateRequest,
    secret?: string
  ): Promise<IssueCertificateResponse> => {
    const payload = secret ? { ...data, secret } : data;
    const res = await api.post("/certificates", payload);
    return res.data;
  },

  revokeCertificate: async (serial: string, secret?: string) => {
    const payload = secret ? { serial, secret } : { serial };
    const res = await api.post("/certificates/revoke", payload);
    return res.data;
  },

  downloadCertificatePackage: (id: string) => {
    return `/api/certificates/${encodeURIComponent(id)}/download`;
  },

  // -----------------------------------------------------
  // AUDIT LOG
  // -----------------------------------------------------

  getAuditLog: async (params?: {
    from?: string;
    to?: string;
    action?: string;
    user?: string;
  }): Promise<{ items: AuditEvent[] }> => {
    const res = await api.get("/audit", { params });
    return res.data;
  },
};

export default apiClient;
