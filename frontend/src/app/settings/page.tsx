'use client';

import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';

// Tato stránka je nyní pouze informační, protože backend neposkytuje CA settings.

export default function SettingsPage() {
  return (
    <div className="page page-settings space-y-6">

      <section className="card">
        <div className="card-header">
          <h2 className="card-header-title">Certificate Authority Information</h2>
        </div>

        <div className="card-body space-y-4">
          <p className="text-gray-700">
            This Certificate Authority is managed automatically by the backend.
            No manual configuration is required or available through the UI.
          </p>

          <p className="text-gray-700">
            Provisioners, certificates, and audit logs can be managed using the
            dedicated sections in the navigation menu.
          </p>

          <p className="text-gray-700">
            Future versions may include editable CA settings once backend support
            is implemented.
          </p>
        </div>
      </section>

    </div>
  );
}
