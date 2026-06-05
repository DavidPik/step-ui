'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import React from 'react';

const NAV_ITEMS = [
  { href: '/', label: 'Dashboard' },
  { href: '/provisioners', label: 'Provisioners' },
  { href: '/certificates', label: 'Certificates' },
  { href: '/settings', label: 'CA Settings' },
  { href: '/audit-log', label: 'Audit Log' },
];

const ACTIVE_PROVISIONER = 'Demo-ACME';

export function Navigation() {
  const pathname = usePathname() || '/';

  return (
    <header className="app-header">
      <nav className="app-nav">
        <div className="app-nav-left">
          {NAV_ITEMS.map((item) => {
            const isActive =
              item.href === '/'
                ? pathname === '/'
                : pathname.startsWith(item.href);

            return (
              <Link
                key={item.href}
                href={item.href}
                className={
                  'app-nav-link' + (isActive ? ' app-nav-link-active' : '')
                }
              >
                {item.label}
              </Link>
            );
          })}
        </div>
        <div className="app-nav-right">
          <span className="app-nav-provisioner-label">Active Provisioner:</span>
          <span className="app-nav-provisioner-value">
            {ACTIVE_PROVISIONER}
          </span>
        </div>
      </nav>
    </header>
  );
}
