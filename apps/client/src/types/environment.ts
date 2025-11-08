import type { MorphSnapshotId } from "@cmux/shared";

export type EnvVar = {
  id: string;
  name: string;
  value: string;
  isSecret: boolean;
};

const generateId = (): string => {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
};

export const ensureInitialEnvVars = (initial?: EnvVar[]): EnvVar[] => {
  const base = (initial ?? []).map((item) => ({
    id: item.id || generateId(),
    name: item.name,
    value: item.value,
    isSecret: item.isSecret ?? true,
  }));
  if (base.length === 0) {
    return [{ id: generateId(), name: "", value: "", isSecret: true }];
  }
  const last = base[base.length - 1];
  if (!last || last.name.trim().length > 0 || last.value.trim().length > 0) {
    base.push({ id: generateId(), name: "", value: "", isSecret: true });
  }
  return base;
};

export interface EnvironmentConfigDraft {
  envName: string;
  envVars: EnvVar[];
  maintenanceScript: string;
  devScript: string;
  exposedPorts: string;
}

export interface EnvironmentDraftMetadata {
  selectedRepos: string[];
  instanceId?: string;
  snapshotId?: MorphSnapshotId;
}

export const createEmptyEnvironmentConfig = (): EnvironmentConfigDraft => ({
  envName: "",
  envVars: ensureInitialEnvVars(),
  maintenanceScript: "",
  devScript: "",
  exposedPorts: "",
});
