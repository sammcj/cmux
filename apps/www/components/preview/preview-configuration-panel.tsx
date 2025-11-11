"use client";

import { useCallback, useEffect, useMemo, useState, useTransition } from "react";
import { Loader2, RefreshCcw, Search, Shield, Upload } from "lucide-react";

const browserOptions = [
  { value: "chromium", label: "Chromium" },
  { value: "firefox", label: "Firefox" },
  { value: "webkit", label: "Webkit" },
] as const;

type BrowserProfile = (typeof browserOptions)[number]["value"];

type PreviewConfigState = {
  id: string | null;
  repoFullName: string;
  repoDefaultBranch: string;
  devScript: string;
  maintenanceScript: string;
  browserProfile: BrowserProfile;
  morphSnapshotId?: string | null;
};

type PreviewRunSummary = {
  id: string;
  prNumber: number;
  prUrl: string;
  headSha: string;
  status: "pending" | "running" | "completed" | "failed" | "skipped";
  stateReason?: string | null;
  createdAt: number;
  updatedAt: number;
};

type RepoSearchResult = {
  full_name: string;
  private: boolean;
  updated_at?: string | null;
};

type PreviewConfigurationPanelProps = {
  teamSlugOrId: string;
  hasGithubAppInstallation: boolean;
  initialConfig: PreviewConfigState | null;
  initialEnvVars: string;
  initialRuns: PreviewRunSummary[];
};

export function PreviewConfigurationPanel({
  teamSlugOrId,
  hasGithubAppInstallation,
  initialConfig,
  initialEnvVars,
  initialRuns,
}: PreviewConfigurationPanelProps) {
  const [config, setConfig] = useState<PreviewConfigState>(
    initialConfig ?? {
      id: null,
      repoFullName: "",
      repoDefaultBranch: "main",
      devScript: "",
      maintenanceScript: "",
      browserProfile: "chromium",
    },
  );
  const [envVars, setEnvVars] = useState(initialEnvVars);
  const [runs, setRuns] = useState<PreviewRunSummary[]>(initialRuns);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isSaving, startSaveTransition] = useTransition();
  const [isInstallingApp, setIsInstallingApp] = useState(false);

  const hasConfig = useMemo(() => Boolean(config.id), [config.id]);

  useEffect(() => {
    if (initialConfig) {
      setConfig(initialConfig);
    }
  }, [initialConfig]);

  useEffect(() => {
    setEnvVars(initialEnvVars);
  }, [initialEnvVars]);

  const refreshRuns = useCallback(async () => {
    if (!config.id) {
      return;
    }
    try {
      const params = new URLSearchParams({ teamSlugOrId });
      const response = await fetch(
        `/api/preview/configs/${config.id}/runs?${params.toString()}`,
      );
      if (!response.ok) {
        throw new Error(await response.text());
      }
      const payload = (await response.json()) as { runs: PreviewRunSummary[] };
      setRuns(payload.runs);
    } catch (error) {
      console.error("Failed to refresh preview runs", error);
    }
  }, [config.id, teamSlugOrId]);

  const handleSave = useCallback(() => {
    setStatusMessage(null);
    setErrorMessage(null);
    startSaveTransition(async () => {
      try {
        const response = await fetch("/api/preview/configs", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            previewConfigId: config.id ?? undefined,
            teamSlugOrId,
            repoFullName: config.repoFullName.trim(),
            repoDefaultBranch: config.repoDefaultBranch.trim(),
            devScript: config.devScript,
            maintenanceScript: config.maintenanceScript,
            browserProfile: config.browserProfile,
            envVarsContent: envVars,
          }),
        });
        if (!response.ok) {
          throw new Error(await response.text());
        }
        const saved = (await response.json()) as PreviewConfigState & {
          hasEnvVars: boolean;
        };
        setConfig((prev) => ({
          ...prev,
          ...saved,
          id: saved.id,
        }));
        setStatusMessage("Configuration saved");
        await refreshRuns();
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to save configuration";
        setErrorMessage(message);
      }
    });
  }, [config, teamSlugOrId, envVars, refreshRuns]);

  const handleInstallGithubApp = useCallback(async () => {
    setIsInstallingApp(true);
    setErrorMessage(null);
    try {
      const response = await fetch("/api/integrations/github/install-state", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          teamSlugOrId,
          returnUrl: window.location.href,
        }),
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      const payload = (await response.json()) as { state: string };
      const githubAppSlug = process.env.NEXT_PUBLIC_GITHUB_APP_SLUG;
      if (!githubAppSlug) {
        throw new Error("GitHub App slug is not configured");
      }
      const url = new URL(`https://github.com/apps/${githubAppSlug}/installations/new`);
      url.searchParams.set("state", payload.state);
      window.location.href = url.toString();
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to start GitHub App install";
      setErrorMessage(message);
    } finally {
      setIsInstallingApp(false);
    }
  }, [teamSlugOrId]);

  return (
    <div className="space-y-10">
      <div className="rounded-2xl border border-neutral-800 bg-neutral-950/60 p-6 shadow-xl shadow-black/30">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-sm text-neutral-400">Repository</p>
            <input
              type="text"
              spellCheck={false}
              placeholder="acme/widgets"
              value={config.repoFullName}
              onChange={(event) =>
                setConfig((prev) => ({ ...prev, repoFullName: event.target.value }))
              }
              className="mt-2 w-full rounded-lg border border-neutral-700 bg-neutral-900 px-3 py-2 font-mono text-sm text-white placeholder:text-neutral-500 focus:border-sky-500 focus:outline-none"
            />
            <p className="mt-2 text-xs text-neutral-500">
              Provide the GitHub owner/name combination you want cmux Preview to watch.
            </p>
          </div>
          <div className="w-full md:w-auto">
            <label className="text-sm text-neutral-400">Browser</label>
            <select
              value={config.browserProfile}
              onChange={(event) =>
                setConfig((prev) => ({
                  ...prev,
                  browserProfile: event.target.value as BrowserProfile,
                }))
              }
              className="mt-2 w-full rounded-lg border border-neutral-700 bg-neutral-900 px-3 py-2 text-sm text-white focus:border-sky-500 focus:outline-none md:w-52"
            >
              {browserOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="mt-6 rounded-xl border border-neutral-800 bg-neutral-900/60 p-4">
          <p className="text-sm font-medium text-neutral-200">Scripts</p>
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-xs uppercase tracking-wide text-neutral-400">
                Maintenance Script
              </label>
              <textarea
                value={config.maintenanceScript}
                onChange={(event) =>
                  setConfig((prev) => ({
                    ...prev,
                    maintenanceScript: event.target.value,
                  }))
                }
                placeholder="pnpm install && pnpm db:pull"
                className="mt-2 h-32 w-full rounded-lg border border-neutral-700 bg-neutral-950 px-3 py-2 text-sm text-white placeholder:text-neutral-500 focus:border-sky-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="text-xs uppercase tracking-wide text-neutral-400">
                Dev Script
              </label>
              <textarea
                value={config.devScript}
                onChange={(event) =>
                  setConfig((prev) => ({ ...prev, devScript: event.target.value }))
                }
                placeholder="pnpm dev -- --host 0.0.0.0"
                className="mt-2 h-32 w-full rounded-lg border border-neutral-700 bg-neutral-950 px-3 py-2 text-sm text-white placeholder:text-neutral-500 focus:border-sky-500 focus:outline-none"
              />
            </div>
          </div>
        </div>

        <div className="mt-6">
          <label className="text-sm text-neutral-400">Environment Variables</label>
          <textarea
            value={envVars}
            onChange={(event) => setEnvVars(event.target.value)}
            placeholder="NEXT_PUBLIC_API_URL=..."
            className="mt-2 h-40 w-full rounded-lg border border-neutral-700 bg-neutral-950 px-3 py-2 font-mono text-sm text-white placeholder:text-neutral-500 focus:border-sky-500 focus:outline-none"
          />
        </div>

        <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-sm text-neutral-400">
            {statusMessage && <span className="text-emerald-400">{statusMessage}</span>}
            {errorMessage && <span className="text-red-400">{errorMessage}</span>}
          </div>
          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={handleSave}
              disabled={isSaving || !config.repoFullName.trim()}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-semibold text-black transition hover:bg-neutral-200 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isSaving ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Saving...
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4" />
                  Save configuration
                </>
              )}
            </button>
            <button
              type="button"
              onClick={refreshRuns}
              disabled={!hasConfig}
              className="inline-flex items-center justify-center gap-2 rounded-lg border border-neutral-700 px-4 py-2 text-sm font-semibold text-white transition hover:border-neutral-500 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <RefreshCcw className="h-4 w-4" />
              Refresh runs
            </button>
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-neutral-800 bg-neutral-950/60 p-6 shadow-xl shadow-black/30">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-neutral-200">GitHub App</p>
          <span className="text-xs text-neutral-400">
            {hasGithubAppInstallation ? "Connected" : "Not installed"}
          </span>
        </div>
        <p className="mt-2 text-sm text-neutral-400">
          Install the cmux GitHub App on your organization or user account so preview runs can watch every PR.
        </p>
        <button
          type="button"
          onClick={handleInstallGithubApp}
          disabled={isInstallingApp}
          className="mt-4 inline-flex items-center justify-center gap-2 rounded-lg border border-neutral-700 px-4 py-2 text-sm font-semibold text-white transition hover:border-neutral-500 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isInstallingApp ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Redirecting to GitHub...
            </>
          ) : (
            <>
              <Shield className="h-4 w-4" />
              Install cmux GitHub App
            </>
          )}
        </button>
      </div>

      <RepoSearch teamSlugOrId={teamSlugOrId} onSelectRepo={(repo) =>
        setConfig((prev) => ({ ...prev, repoFullName: repo }))
      } />

      <div className="rounded-2xl border border-neutral-800 bg-neutral-950/60 p-6 shadow-xl shadow-black/30">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-neutral-200">Recent Runs</p>
          {hasConfig && (
            <button
              type="button"
              onClick={refreshRuns}
              className="text-xs text-neutral-400 hover:text-neutral-200"
            >
              Refresh
            </button>
          )}
        </div>
        {runs.length === 0 ? (
          <p className="mt-4 text-sm text-neutral-400">
            No preview runs have been recorded yet. We'll trigger one automatically the next time a pull request is opened.
          </p>
        ) : (
          <ul className="mt-4 space-y-3">
            {runs.map((run) => (
              <li
                key={run.id}
                className="rounded-xl border border-neutral-800 bg-neutral-900/40 p-4"
              >
                <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                  <div>
                    <p className="text-sm font-semibold text-white">
                      PR #{run.prNumber}
                    </p>
                    <a
                      href={run.prUrl}
                      className="text-xs text-sky-400 hover:underline"
                      target="_blank"
                      rel="noreferrer"
                    >
                      {run.prUrl}
                    </a>
                  </div>
                  <div className="text-xs uppercase tracking-wide text-neutral-400">
                    Status: <span className="text-white">{run.status}</span>
                  </div>
                </div>
                {run.stateReason && (
                  <p className="mt-2 text-xs text-red-400">{run.stateReason}</p>
                )}
                <p className="mt-2 text-xs text-neutral-500">
                  Updated {new Date(run.updatedAt).toLocaleString()}
                </p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

type RepoSearchProps = {
  teamSlugOrId: string;
  onSelectRepo: (repoFullName: string) => void;
};

function RepoSearch({ teamSlugOrId, onSelectRepo }: RepoSearchProps) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<RepoSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = useCallback(async () => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    setIsSearching(true);
    setError(null);
    try {
      const params = new URLSearchParams({ team: teamSlugOrId, search: query.trim() });
      const response = await fetch(`/api/integrations/github/repos?${params}`);
      if (!response.ok) {
        throw new Error(await response.text());
      }
      const payload = (await response.json()) as { repos: RepoSearchResult[] };
      setResults(payload.repos);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load repositories";
      setError(message);
    } finally {
      setIsSearching(false);
    }
  }, [query, teamSlugOrId]);

  return (
    <div className="rounded-2xl border border-neutral-800 bg-neutral-950/60 p-6 shadow-xl shadow-black/30">
      <div className="flex flex-col gap-3">
        <label className="text-sm font-medium text-neutral-200">Find a repository</label>
        <div className="flex flex-col gap-3 md:flex-row">
          <div className="flex flex-1 items-center gap-2 rounded-lg border border-neutral-700 bg-neutral-900 px-3">
            <Search className="h-4 w-4 text-neutral-500" />
            <input
              type="text"
              spellCheck={false}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search installations"
              className="flex-1 border-none bg-transparent text-sm text-white placeholder:text-neutral-500 focus:outline-none"
            />
          </div>
          <button
            type="button"
            onClick={handleSearch}
            disabled={isSearching}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-neutral-700 px-4 py-2 text-sm font-semibold text-white transition hover:border-neutral-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {isSearching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
            Search
          </button>
        </div>
        {error && <p className="text-xs text-red-400">{error}</p>}
        {results.length > 0 && (
          <ul className="space-y-2">
            {results.map((repo) => (
              <li
                key={repo.full_name}
                className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-900/60 px-3 py-2"
              >
                <div>
                  <p className="text-sm font-medium text-white">{repo.full_name}</p>
                  {repo.updated_at && (
                    <p className="text-xs text-neutral-500">
                      Updated {new Date(repo.updated_at).toLocaleDateString()}
                    </p>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => onSelectRepo(repo.full_name)}
                  className="text-xs font-semibold text-sky-400 hover:text-sky-300"
                >
                  Use repo
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
