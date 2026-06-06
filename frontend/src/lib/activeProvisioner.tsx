"use client";

import { createContext, useContext, useEffect, useState } from "react";
import { apiClient } from "./api";

type ProvisionerStatusValue = "online" | "offline" | "error" | "unknown";

interface ActiveProvisionerContextType {
  activeProvisioner: string | null;
  activeProvisionerStatus: ProvisionerStatusValue;
  setActiveProvisioner: (name: string | null) => void;
  refreshActiveProvisionerStatus: () => void;
}

const ActiveProvisionerContext = createContext<ActiveProvisionerContextType>({
  activeProvisioner: null,
  activeProvisionerStatus: "unknown",
  setActiveProvisioner: () => {},
  refreshActiveProvisionerStatus: () => {},
});

export function ActiveProvisionerProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [activeProvisioner, setActiveProvisionerState] = useState<string | null>(
    null
  );
  const [activeProvisionerStatus, setActiveProvisionerStatus] =
    useState<ProvisionerStatusValue>("unknown");

  // načtení z localStorage při startu
  useEffect(() => {
    const stored = window.localStorage.getItem("activeProvisioner");
    if (stored) {
      setActiveProvisionerState(stored);
    }
  }, []);

  // kdykoliv se změní jméno, zkus načíst stav z backendu
  useEffect(() => {
    if (!activeProvisioner) {
      setActiveProvisionerStatus("unknown");
      return;
    }
    refreshStatus(activeProvisioner);
  }, [activeProvisioner]);

  function setActiveProvisioner(name: string | null) {
    setActiveProvisionerState(name);
    if (name) {
      window.localStorage.setItem("activeProvisioner", name);
    } else {
      window.localStorage.removeItem("activeProvisioner");
    }
  }

  async function refreshStatus(name: string) {
    try {
      const res = await apiClient.getProvisionerStatus(name);
      setActiveProvisionerStatus(res.status as ProvisionerStatusValue);
    } catch {
      setActiveProvisionerStatus("offline");
    }
  }

  function refreshActiveProvisionerStatus() {
    if (activeProvisioner) {
      refreshStatus(activeProvisioner);
    }
  }

  return (
    <ActiveProvisionerContext.Provider
      value={{
        activeProvisioner,
        activeProvisionerStatus,
        setActiveProvisioner,
        refreshActiveProvisionerStatus,
      }}
    >
      {children}
    </ActiveProvisionerContext.Provider>
  );
}

export function useActiveProvisioner() {
  return useContext(ActiveProvisionerContext);
}
