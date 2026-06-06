import axios from "axios";
import type {
  Provisioner,
  CertificateItem,
  CertificateDetail,
  IssueCertificateRequest,
  IssueCertificateResponse,
  AuditEvent,
  ListResponse,
} from "./types";

const api = axios.create({
  baseURL: "/api",
  headers: {
    "Content-Type": "application/json",
  },
});

export const apiClient = {
  //
  // PROVISIONERS
  //
  listProvisioners: async (): Promise<ListResponse<Provisioner>> => {
    const res = await api.get("/provisioners");
    return res.data;
  },

  createProvisioner: async (data: {
    name: string;
    type: string;
    jwk?: string;
    acme_directories?: string[];
    secret?: string;
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

  getProvisionerStatuses: async () => {
    const res = await api.get("/provisioners/status");
    return res.data;
  },

  getProvisionerStatus: async (name: string) => {
    const res = await api.get(`/provisioners/${encodeURIComponent(name)}/status`);
    return res.data;
  },

  //
  // CERTIFICATES
  //
  listCertificates: async (): Promise<ListResponse<CertificateItem>> => {
    const res = await api.get("/certificates");
    return res.data;
  },

  getCertificate: async (id: string): Promise<CertificateDetail> => {
    const res = await api.get(`/certificates/${encodeURIComponent(id)}`);
    return res.data;
  },

  issueCertificate: async (
    data: IssueCertificateRequest,
    secret?: string
  ): Promise<IssueCertificateResponse> => {
    const payload = secret ? { ...data, secret } : data;
    const res = await api.post("/certificates/issue", payload);
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

  //
  // AUDIT LOG
  //
  getAuditLog: async (params?: {
    from?: string;
    to?: string;
    action?: string;
    user?: string;
  }): Promise<ListResponse<AuditEvent>> => {
    const res = await api.get("/audit", { params });
    return res.data;
  },
};

export default apiClient;
