import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// ---------------------------------------------------------
// Class name merge helper (pro UI komponenty)
// ---------------------------------------------------------

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// ---------------------------------------------------------
// Date helpers
// ---------------------------------------------------------

export function formatDate(date: string | Date) {
  return new Date(date).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function getDaysUntilExpiry(notAfter: string) {
  const expiry = new Date(notAfter);
  const now = new Date();
  const diffTime = expiry.getTime() - now.getTime();
  return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
}

export function getExpiryStatus(notAfter: string) {
  const days = getDaysUntilExpiry(notAfter);
  if (days < 0) return "expired";
  if (days <= 7) return "critical";
  if (days <= 30) return "warning";
  return "good";
}

// ---------------------------------------------------------
// File download helpers
// ---------------------------------------------------------

export function downloadFile(data: string, filename: string, mimeType: string) {
  const blob = new Blob([data], { type: mimeType });
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}

export function downloadBase64File(
  base64: string,
  filename: string,
  mimeType: string
) {
  const blob = new Blob(
    [Uint8Array.from(atob(base64), (c) => c.charCodeAt(0))],
    { type: mimeType }
  );
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
