// ------------------------------------------------------------
// ISSUE CERTIFICATE DIALOG
// ------------------------------------------------------------

function IssueCertificateDialog({
  open,
  onClose,
  onIssued,
}: {
  open: boolean;
  onClose: () => void;
  onIssued: () => void;
}) {
  const [cn, setCn] = useState("");
  const [dns, setDns] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleIssue() {
    setLoading(true);
    try {
      await apiClient.issueCertificate({
        common_name: cn,
        dns_names: dns.split(",").map((s) => s.trim()).filter(Boolean),
      });
      onIssued();
      onClose();
    } catch (err) {
      console.error("Failed to issue certificate", err);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-lg">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Issue Certificate</h3>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Common Name</Label>
            <Input value={cn} onChange={(e) => setCn(e.target.value)} />
          </div>

          <div>
            <Label>DNS Names (comma separated)</Label>
            <Input value={dns} onChange={(e) => setDns(e.target.value)} />
          </div>
        </div>

        <DialogFooter>
          <Button onClick={onClose}>Cancel</Button>
          <Button onClick={handleIssue} disabled={loading}>
            {loading ? "Issuing…" : "Issue"}
          </Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}

// ------------------------------------------------------------
// SIGN CSR DIALOG — REMOVED (backend does not support CSR)
// ------------------------------------------------------------

// ------------------------------------------------------------
// CERTIFICATE DETAIL DIALOG
// ------------------------------------------------------------

function CertificateDetailDialog({
  open,
  cert,
  onClose,
  onRevoke,
}: {
  open: boolean;
  cert: CertificateDetail | null;
  onClose: () => void;
  onRevoke: () => void;
}) {
  if (!cert) return null;

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-2xl">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Certificate Details</h3>
        </DialogHeader>

        <div className="space-y-2">
          <div>
            <strong>Common Name:</strong> {cert.common_name}
          </div>
          <div>
            <strong>Serial:</strong> {cert.serial}
          </div>
          <div>
            <strong>Issued:</strong>{" "}
            {new Date(cert.not_before).toLocaleString()}
          </div>
          <div>
            <strong>Expires:</strong>{" "}
            {new Date(cert.not_after).toLocaleString()}
          </div>

          <div>
            <strong>Download:</strong>
            <div className="flex gap-4 mt-1">
              <Button
                onClick={() =>
                  downloadFile(
                    cert.certificate_pem,
                    `${cert.common_name}.crt`,
                    "application/x-pem-file"
                  )
                }
              >
                Certificate (PEM)
              </Button>

              <Button
                onClick={() =>
                  downloadFile(
                    cert.ca_bundle_pem,
                    `${cert.common_name}-chain.crt`,
                    "application/x-pem-file"
                  )
                }
              >
                Chain (PEM)
              </Button>

              <Button
                onClick={() => {
                  window.location.href =
                    apiClient.downloadCertificatePackage(cert.id);
                }}
              >
                ZIP Package
              </Button>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button onClick={onRevoke}>Revoke</Button>
          <Button onClick={onClose}>Close</Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}

// ------------------------------------------------------------
// REVOKE CERTIFICATE DIALOG
// ------------------------------------------------------------

function RevokeCertificateDialog({
  open,
  cert,
  onClose,
  onRevoked,
}: {
  open: boolean;
  cert: CertificateDetail | null;
  onClose: () => void;
  onRevoked: () => void;
}) {
  const [loading, setLoading] = useState(false);

  if (!cert) return null;

  async function handleRevoke() {
    setLoading(true);
    try {
      await apiClient.revokeCertificate(cert.serial);
      onRevoked();
      onClose();
    } catch (err) {
      console.error("Failed to revoke certificate", err);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <div className="bg-white p-6 rounded shadow max-w-lg">
        <DialogHeader>
          <h3 className="text-lg font-semibold">Revoke Certificate</h3>
        </DialogHeader>

        <p>
          Are you sure you want to revoke certificate{" "}
          <strong>{cert.common_name}</strong>?
        </p>

        <DialogFooter>
          <Button onClick={onClose}>Cancel</Button>
          <Button onClick={handleRevoke} disabled={loading}>
            {loading ? "Revoking…" : "Revoke"}
          </Button>
        </DialogFooter>
      </div>
    </Dialog>
  );
}
