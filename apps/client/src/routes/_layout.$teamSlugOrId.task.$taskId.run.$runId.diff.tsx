import { FloatingPane } from "@/components/floating-pane";
import { type GitDiffViewerProps } from "@/components/git-diff-viewer";
import { RunDiffSection } from "@/components/RunDiffSection";
import { RunScreenshotGallery } from "@/components/RunScreenshotGallery";
import { TaskDetailHeader } from "@/components/task-detail-header";
import { useSocket } from "@/contexts/socket/use-socket";
import { normalizeGitRef } from "@/lib/refWithOrigin";
import { gitDiffQueryOptions } from "@/queries/git-diff";
import { parseReviewHeatmap, type ReviewHeatmapLine } from "@/lib/heatmap";
import { api } from "@cmux/convex/api";
import type { Doc } from "@cmux/convex/dataModel";
import type { CreateLocalWorkspaceResponse, ReplaceDiffEntry } from "@cmux/shared";
import { typedZid } from "@cmux/shared/utils/typed-zid";
import { convexQuery } from "@convex-dev/react-query";
import { useQuery as useRQ, useMutation as useRQMutation } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useQuery } from "convex/react";
import {
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { toast } from "sonner";
import z from "zod";
import { useCombinedWorkflowData, WorkflowRunsSection } from "@/components/WorkflowRunsSection";
import { convexQueryClient } from "@/contexts/convex/convex-query-client";
import { postApiCodeReviewStartMutation } from "@cmux/www-openapi-client/react-query";

/** Convert ReplaceDiffEntry[] to the format expected by heatmap API */
function convertDiffsToFileDiffs(
  diffs: ReplaceDiffEntry[]
): Array<{ filePath: string; diffText: string }> {
  return diffs
    .filter((entry) => entry.patch && !entry.isBinary)
    .map((entry) => ({
      filePath: entry.filePath,
      diffText: entry.patch!,
    }));
}

const paramsSchema = z.object({
  taskId: typedZid("tasks"),
  runId: typedZid("taskRuns"),
});

const gitDiffViewerClassNames: GitDiffViewerProps["classNames"] = {
  fileDiffRow: {
    button: "top-[96px] md:top-[56px]",
  },
};

type DiffControls = Parameters<
  NonNullable<GitDiffViewerProps["onControlsChange"]>
>[0];

function WorkflowRunsWrapper({
  teamSlugOrId,
  repoFullName,
  prNumber,
  headSha,
  checksExpandedByRepo,
  setChecksExpandedByRepo,
}: {
  teamSlugOrId: string;
  repoFullName: string;
  prNumber: number;
  headSha?: string;
  checksExpandedByRepo: Record<string, boolean | null>;
  setChecksExpandedByRepo: React.Dispatch<React.SetStateAction<Record<string, boolean | null>>>;
}) {
  const workflowData = useCombinedWorkflowData({
    teamSlugOrId,
    repoFullName,
    prNumber,
    headSha,
  });

  // Auto-expand if there are failures (only on initial load)
  const hasAnyFailure = useMemo(() => {
    return workflowData.allRuns.some(
      (run) =>
        run.conclusion === "failure" ||
        run.conclusion === "timed_out" ||
        run.conclusion === "action_required"
    );
  }, [workflowData.allRuns]);

  const isExpanded = checksExpandedByRepo[repoFullName] ?? hasAnyFailure;

  return (
    <WorkflowRunsSection
      allRuns={workflowData.allRuns}
      isLoading={workflowData.isLoading}
      isExpanded={isExpanded}
      onToggle={() => {
        setChecksExpandedByRepo((prev) => ({
          ...prev,
          [repoFullName]: !isExpanded,
        }));
      }}
    />
  );
}

export const Route = createFileRoute(
  "/_layout/$teamSlugOrId/task/$taskId/run/$runId/diff",
)({
  component: RunDiffPage,
  params: {
    parse: paramsSchema.parse,
    stringify: (params) => {
      return {
        taskId: params.taskId,
        runId: params.runId,
      };
    },
  },
  loader: (opts) => {
    const { runId } = opts.params;

    convexQueryClient.convexClient.prewarmQuery({
      query: api.taskRuns.getRunDiffContext,
      args: { teamSlugOrId: opts.params.teamSlugOrId, taskId: opts.params.taskId, runId },
    });

    void opts.context.queryClient
      .ensureQueryData(
        convexQuery(api.taskRuns.getRunDiffContext, {
          teamSlugOrId: opts.params.teamSlugOrId,
          taskId: opts.params.taskId,
          runId,
        }),
      )
      .then(async (context) => {
        if (!context) {
          return;
        }

        const { task, taskRuns, branchMetadataByRepo } = context;

        if (task) {
          opts.context.queryClient.setQueryData(
            convexQuery(api.tasks.getById, {
              teamSlugOrId: opts.params.teamSlugOrId,
              id: opts.params.taskId,
            }).queryKey,
            task,
          );
        }

        if (taskRuns) {
          opts.context.queryClient.setQueryData(
            convexQuery(api.taskRuns.getByTask, {
              teamSlugOrId: opts.params.teamSlugOrId,
              taskId: opts.params.taskId,
            }).queryKey,
            taskRuns,
          );
        }

        const selectedTaskRun = taskRuns.find((run) => run._id === runId);
        if (!task || !selectedTaskRun?.newBranch) {
          return;
        }

        const trimmedProjectFullName = task.projectFullName?.trim();
        const targetRepos = new Set<string>();
        for (const repo of selectedTaskRun.environment?.selectedRepos ?? []) {
          const trimmed = repo?.trim();
          if (trimmed) {
            targetRepos.add(trimmed);
          }
        }
        if (trimmedProjectFullName) {
          targetRepos.add(trimmedProjectFullName);
        }

        if (targetRepos.size === 0) {
          return;
        }

        const baseRefForDiff = normalizeGitRef(task.baseBranch || "main");
        const headRefForDiff = normalizeGitRef(selectedTaskRun.newBranch);
        if (!headRefForDiff || !baseRefForDiff) {
          return;
        }

        const metadataForPrimaryRepo = trimmedProjectFullName
          ? branchMetadataByRepo?.[trimmedProjectFullName]
          : undefined;
        const baseBranchMeta = metadataForPrimaryRepo?.find(
          (branch) => branch.name === task.baseBranch,
        );

        const prefetches = Array.from(targetRepos).map(async (repoFullName) => {
          const metadata =
            trimmedProjectFullName && repoFullName === trimmedProjectFullName
              ? baseBranchMeta
              : undefined;

          return opts.context.queryClient
            .ensureQueryData(
              gitDiffQueryOptions({
                baseRef: baseRefForDiff,
                headRef: headRefForDiff,
                repoFullName,
                lastKnownBaseSha: metadata?.lastKnownBaseSha,
                lastKnownMergeCommitSha: metadata?.lastKnownMergeCommitSha,
              }),
            )
            .catch(() => undefined);
        });

        await Promise.all(prefetches);
      })
      .catch(() => undefined);

    return undefined;
  },
});

function RunDiffPage() {
  const { taskId, teamSlugOrId, runId } = Route.useParams();
  const [diffControls, setDiffControls] = useState<DiffControls | null>(null);
  const { socket } = useSocket();
  const task = useQuery(api.tasks.getById, {
    teamSlugOrId,
    id: taskId,
  });
  const taskRuns = useQuery(api.taskRuns.getByTask, {
    teamSlugOrId,
    taskId,
  });
  const selectedRun = useMemo(() => {
    return taskRuns?.find((run) => run._id === runId);
  }, [runId, taskRuns]);

  // Heatmap review state - automatically fetched via streaming Convex subscription
  const [comparisonSlug, setComparisonSlug] = useState<string | null>(null);

  // Query workspace settings for heatmap configuration
  const workspaceSettingsQuery = useRQ({
    ...convexQuery(api.workspaceSettings.get, { teamSlugOrId }),
    enabled: Boolean(teamSlugOrId),
  });
  const workspaceSettings = workspaceSettingsQuery.data as {
    heatmapThreshold?: number;
    heatmapModel?: string;
    heatmapTooltipLanguage?: string;
  } | null | undefined;
  const heatmapThreshold = workspaceSettings?.heatmapThreshold ?? 0;
  const heatmapModel = workspaceSettings?.heatmapModel;
  const heatmapTooltipLanguage = workspaceSettings?.heatmapTooltipLanguage;

  // Code review mutation to start the heatmap job
  const codeReviewMutation = useRQMutation({
    ...postApiCodeReviewStartMutation(),
    onSuccess: (data) => {
      console.log("[heatmap] Mutation success", {
        jobId: data.job?.jobId,
        state: data.job?.state,
        comparisonSlug: data.job?.comparisonSlug,
      });

      // Store comparison slug for subscription (if not already set)
      if (data.job?.comparisonSlug) {
        setComparisonSlug(data.job.comparisonSlug);
      }
    },
    onError: (error) => {
      console.error("[heatmap] Mutation failed:", error);
    },
  });

  // Use ref to avoid mutation object in dependency arrays (prevents infinite loops)
  const codeReviewMutationRef = useRef(codeReviewMutation);
  codeReviewMutationRef.current = codeReviewMutation;

  const runDiffContextQuery = useRQ({
    ...convexQuery(api.taskRuns.getRunDiffContext, {
      teamSlugOrId,
      taskId,
      runId,
    }),
    enabled: Boolean(teamSlugOrId && taskId && runId),
  });

  const screenshotSets = runDiffContextQuery.data?.screenshotSets ?? [];
  const screenshotSetsLoading =
    runDiffContextQuery.isLoading && screenshotSets.length === 0;

  // Get PR information from the selected run
  const pullRequests = useMemo(() => {
    return selectedRun?.pullRequests?.filter(
      (pr) => pr.number !== undefined && pr.number !== null
    ) as Array<{ repoFullName: string; number: number; url?: string }> | undefined;
  }, [selectedRun]);

  // Track expanded state for each PR's checks
  const [checksExpandedByRepo, setChecksExpandedByRepo] = useState<Record<string, boolean | null>>({});

  const expandAllChecks = useCallback(() => {
    if (!pullRequests) return;
    const newState: Record<string, boolean | null> = {};
    for (const pr of pullRequests) {
      newState[pr.repoFullName] = true;
    }
    setChecksExpandedByRepo(newState);
  }, [pullRequests]);

  const collapseAllChecks = useCallback(() => {
    if (!pullRequests) return;
    const newState: Record<string, boolean | null> = {};
    for (const pr of pullRequests) {
      newState[pr.repoFullName] = false;
    }
    setChecksExpandedByRepo(newState);
  }, [pullRequests]);

  const environmentRepos = useMemo(() => {
    const repos = selectedRun?.environment?.selectedRepos ?? [];
    const trimmed = repos
      .map((repo) => repo?.trim())
      .filter((repo): repo is string => Boolean(repo));
    return Array.from(new Set(trimmed));
  }, [selectedRun]);

  const repoFullNames = useMemo(() => {
    if (task?.projectFullName) {
      return [task.projectFullName];
    }
    return environmentRepos;
  }, [task?.projectFullName, environmentRepos]);

  const [primaryRepo, ...additionalRepos] = repoFullNames;

  const branchMetadataQuery = useRQ({
    ...convexQuery(api.github.getBranchesByRepo, {
      teamSlugOrId,
      repo: primaryRepo ?? "",
    }),
    enabled: Boolean(primaryRepo),
  });

  const branchMetadata = branchMetadataQuery.data as
    | Doc<"branches">[]
    | undefined;

  const baseBranchMetadata = useMemo(() => {
    if (!task?.baseBranch) {
      return undefined;
    }
    return branchMetadata?.find((branch) => branch.name === task.baseBranch);
  }, [branchMetadata, task?.baseBranch]);

  const metadataByRepo = useMemo(() => {
    if (!primaryRepo) return undefined;
    if (!baseBranchMetadata) return undefined;
    const { lastKnownBaseSha, lastKnownMergeCommitSha } = baseBranchMetadata;
    if (!lastKnownBaseSha && !lastKnownMergeCommitSha) {
      return undefined;
    }
    return {
      [primaryRepo]: {
        lastKnownBaseSha: lastKnownBaseSha ?? undefined,
        lastKnownMergeCommitSha: lastKnownMergeCommitSha ?? undefined,
      },
    };
  }, [primaryRepo, baseBranchMetadata]);

  // Fetch diffs for heatmap review (this reuses the cached data from RunDiffSection)
  const baseRefForHeatmap = normalizeGitRef(task?.baseBranch || "main");
  const headRefForHeatmap = normalizeGitRef(selectedRun?.newBranch);
  const diffQueryEnabled = Boolean(primaryRepo) && Boolean(baseRefForHeatmap) && Boolean(headRefForHeatmap);
  const diffQuery = useRQ({
    ...gitDiffQueryOptions({
      repoFullName: primaryRepo ?? "",
      baseRef: baseRefForHeatmap,
      headRef: headRefForHeatmap ?? "",
      lastKnownBaseSha: baseBranchMetadata?.lastKnownBaseSha ?? undefined,
      lastKnownMergeCommitSha: baseBranchMetadata?.lastKnownMergeCommitSha ?? undefined,
    }),
    enabled: diffQueryEnabled,
  });

  // Convert diffs to the format expected by the heatmap API
  const fileDiffsForHeatmap = useMemo(() => {
    if (!diffQuery.data) return undefined;
    return convertDiffsToFileDiffs(diffQuery.data);
  }, [diffQuery.data]);

  // Subscribe to streaming file outputs from Convex
  // IMPORTANT: Use useMemo for query args to prevent new object reference on every render
  // (which would cause Convex to re-subscribe and trigger infinite loops)
  const fileOutputsQueryArgs = useMemo(
    () =>
      comparisonSlug && primaryRepo
        ? {
            teamSlugOrId,
            repoFullName: primaryRepo,
            comparisonSlug,
          }
        : ("skip" as const),
    [teamSlugOrId, primaryRepo, comparisonSlug]
  );
  const fileOutputs = useQuery(
    api.codeReview.listFileOutputsForComparison,
    fileOutputsQueryArgs
  );

  // Derive heatmapByFile from fileOutputs using useMemo (avoids setState infinite loops)
  const heatmapByFile = useMemo(() => {
    if (!fileOutputs || fileOutputs.length === 0) {
      return undefined;
    }
    console.log("[heatmap] Processing file outputs", {
      totalFiles: fileOutputs.length,
      files: fileOutputs.map((o) => o.filePath),
      queryArgs: fileOutputsQueryArgs,
    });
    const heatmapData = new Map<string, ReviewHeatmapLine[]>();
    for (const output of fileOutputs) {
      const parsed = parseReviewHeatmap(output.codexReviewOutput);
      console.log("[heatmap] Parsed file", {
        filePath: output.filePath,
        parsedLines: parsed.length,
        rawLinesCount: Array.isArray((output.codexReviewOutput as {lines?: unknown[]})?.lines)
          ? (output.codexReviewOutput as {lines: unknown[]}).lines.length
          : "unknown",
      });
      if (parsed.length > 0) {
        heatmapData.set(output.filePath, parsed);
      }
    }
    console.log("[heatmap] Final heatmap data", {
      filesWithSuggestions: heatmapData.size,
      totalFilesProcessed: fileOutputs.length,
    });
    return heatmapData.size > 0 ? heatmapData : undefined;
  }, [fileOutputs, fileOutputsQueryArgs]);

  const taskRunId = selectedRun?._id ?? runId;

  const navigate = useNavigate();

  const handleOpenLocalWorkspace = useCallback(() => {
    if (!socket) {
      toast.error("Socket not connected");
      return;
    }

    if (!primaryRepo) {
      toast.error("No repository information available");
      return;
    }

    if (!selectedRun?.newBranch) {
      toast.error("No branch information available");
      return;
    }

    const loadingToast = toast.loading("Creating local workspace...");

    socket.emit(
      "create-local-workspace",
      {
        teamSlugOrId,
        projectFullName: primaryRepo,
        repoUrl: `https://github.com/${primaryRepo}.git`,
        branch: selectedRun.newBranch,
      },
      (response: CreateLocalWorkspaceResponse) => {
        if (response.success && response.workspacePath) {
          toast.success("Workspace created successfully!", {
            id: loadingToast,
            description: `Opening workspace at ${response.workspacePath}`,
          });

          // Navigate to the vscode view for this task run
          if (response.taskRunId) {
            navigate({
              to: "/$teamSlugOrId/task/$taskId/run/$runId/vscode",
              params: {
                teamSlugOrId,
                taskId,
                runId: response.taskRunId,
              },
            });
          }
        } else {
          toast.error(response.error || "Failed to create workspace", {
            id: loadingToast,
          });
        }
      }
    );
  }, [socket, teamSlugOrId, primaryRepo, selectedRun?.newBranch, navigate, taskId]);

  // Handler to trigger heatmap review
  const triggerHeatmapReview = useCallback((
    force = false,
    diffs?: Array<{ filePath: string; diffText: string }>,
    model?: string,
    language?: string
  ) => {
    if (!primaryRepo || !selectedRun?.newBranch) {
      console.warn("[heatmap] Cannot trigger: missing primaryRepo or newBranch", {
        primaryRepo,
        newBranch: selectedRun?.newBranch,
      });
      return;
    }

    // Get the repo owner for comparison context
    const [repoOwner, repoName] = primaryRepo.split("/");
    if (!repoOwner || !repoName) {
      console.warn("[heatmap] Cannot trigger: invalid repo format", { primaryRepo });
      return;
    }

    // Use task.baseBranch or default to "main"
    const baseBranch = task?.baseBranch || "main";
    const githubLink = `https://github.com/${primaryRepo}`;
    const comparisonSlugValue = `${baseBranch}...${selectedRun.newBranch}`;

    console.log("[heatmap] Triggering heatmap review", {
      primaryRepo,
      baseBranch,
      headBranch: selectedRun.newBranch,
      comparisonSlug: comparisonSlugValue,
      force,
      fileDiffsCount: diffs?.length ?? 0,
      heatmapModel: model,
      tooltipLanguage: language,
    });

    // Set comparisonSlug immediately to start subscription
    setComparisonSlug(comparisonSlugValue);

    codeReviewMutationRef.current.mutate({
      body: {
        teamSlugOrId,
        githubLink,
        headCommitRef: selectedRun.newBranch,
        baseCommitRef: baseBranch,
        force,
        comparison: {
          slug: comparisonSlugValue,
          base: {
            owner: repoOwner,
            repo: repoName,
            ref: baseBranch,
            label: `${repoOwner}:${baseBranch}`,
          },
          head: {
            owner: repoOwner,
            repo: repoName,
            ref: selectedRun.newBranch,
            label: `${repoOwner}:${selectedRun.newBranch}`,
          },
        },
        // Pass pre-fetched diffs to avoid re-fetching from GitHub API
        fileDiffs: diffs,
        // Pass heatmap settings from workspace settings
        heatmapModel: model,
        tooltipLanguage: language,
      },
    });
  }, [primaryRepo, selectedRun?.newBranch, task?.baseBranch, teamSlugOrId]);

  // Track if we've already auto-triggered the heatmap review
  const hasAutoTriggeredRef = useRef(false);
  // Track the last triggered branch/commit to detect changes and re-trigger
  const lastTriggeredBranchRef = useRef<string | null>(null);

  // Auto-trigger heatmap review when all required data is available (including diffs and settings)
  // Also re-triggers when the branch changes (e.g., new commits pushed)
  // The backend handles caching - if results already exist, it returns them immediately
  useEffect(() => {
    // Wait for all required data to be ready (including diffs)
    if (!primaryRepo || !selectedRun?.newBranch) {
      return;
    }
    // Wait for diff query to complete (or fail) before triggering
    // This ensures we can pass the pre-fetched diffs to the backend
    if (diffQuery.isLoading) {
      return;
    }
    // Wait for workspace settings to load (to get model and language preferences)
    if (workspaceSettingsQuery.isLoading) {
      return;
    }

    // Build a key that represents the current state (branch + base)
    const baseBranch = task?.baseBranch || "main";
    const currentKey = `${primaryRepo}:${baseBranch}...${selectedRun.newBranch}`;

    // Check if we need to re-trigger (first time or branch changed)
    const shouldTrigger = !hasAutoTriggeredRef.current || lastTriggeredBranchRef.current !== currentKey;
    if (!shouldTrigger) {
      return;
    }

    // Mark as triggered and store the current key
    hasAutoTriggeredRef.current = true;
    lastTriggeredBranchRef.current = currentKey;

    // Compute the comparison slug to start the subscription immediately
    const comparisonSlugValue = `${baseBranch}...${selectedRun.newBranch}`;
    setComparisonSlug(comparisonSlugValue);

    // Trigger the review with pre-fetched diffs and settings (backend will return cached results if available)
    // Force re-run if the branch changed (lastTriggeredBranchRef was different)
    triggerHeatmapReview(false, fileDiffsForHeatmap, heatmapModel, heatmapTooltipLanguage);
  }, [primaryRepo, selectedRun?.newBranch, task?.baseBranch, triggerHeatmapReview, diffQuery.isLoading, fileDiffsForHeatmap, workspaceSettingsQuery.isLoading, heatmapModel, heatmapTooltipLanguage]);

  // 404 if selected run is missing
  if (!selectedRun) {
    return (
      <div className="p-6 text-sm text-neutral-600 dark:text-neutral-300">
        404 – Run not found
      </div>
    );
  }

  const baseRef = normalizeGitRef(task?.baseBranch || "main");
  const headRef = normalizeGitRef(selectedRun.newBranch);
  const hasDiffSources =
    Boolean(primaryRepo) && Boolean(baseRef) && Boolean(headRef);
  const shouldPrefixDiffs = repoFullNames.length > 1;

  // Only show the "Open local workspace" button for regular tasks (not local/cloud workspaces)
  const isWorkspace = task?.isLocalWorkspace || task?.isCloudWorkspace;

  return (
    <FloatingPane>
      <div className="flex h-full min-h-0 flex-col relative isolate">
        <div className="flex-1 min-h-0 overflow-y-auto flex flex-col">
          <TaskDetailHeader
            task={task}
            taskRuns={taskRuns ?? null}
            selectedRun={selectedRun ?? null}
            taskRunId={taskRunId}
            onExpandAll={diffControls?.expandAll}
            onCollapseAll={diffControls?.collapseAll}
            onExpandAllChecks={expandAllChecks}
            onCollapseAllChecks={collapseAllChecks}
            onOpenLocalWorkspace={isWorkspace ? undefined : handleOpenLocalWorkspace}
            teamSlugOrId={teamSlugOrId}
          />
          {task?.text && (
            <div className="mb-2 px-3.5">
              <div className="text-xs text-neutral-600 dark:text-neutral-300">
                <span className="text-neutral-500 dark:text-neutral-400 select-none">
                  Prompt:{" "}
                </span>
                <span className="font-medium">{task.text}</span>
              </div>
            </div>
          )}
          <div className="bg-white dark:bg-neutral-900 flex-1 min-h-0 flex flex-col">
            {pullRequests && pullRequests.length > 0 && (
              <Suspense fallback={null}>
                {pullRequests.map((pr) => (
                  <WorkflowRunsWrapper
                    key={pr.repoFullName}
                    teamSlugOrId={teamSlugOrId}
                    repoFullName={pr.repoFullName}
                    prNumber={pr.number}
                    headSha={undefined}
                    checksExpandedByRepo={checksExpandedByRepo}
                    setChecksExpandedByRepo={setChecksExpandedByRepo}
                  />
                ))}
              </Suspense>
            )}
            {screenshotSetsLoading ? (
              <div className="border-b border-neutral-200 dark:border-neutral-800 bg-neutral-50/60 dark:bg-neutral-950/40 px-3.5 py-3 text-sm text-neutral-500 dark:text-neutral-400">
                Loading screenshots...
              </div>
            ) : (
              <RunScreenshotGallery
                screenshotSets={screenshotSets}
                highlightedSetId={selectedRun?.latestScreenshotSetId ?? null}
              />
            )}
            <div className="flex-1 min-h-0">
              <Suspense
                fallback={
                  <div className="flex h-full items-center justify-center">
                    <div className="text-neutral-500 dark:text-neutral-400 text-sm select-none">
                      Loading diffs...
                    </div>
                  </div>
                }
              >
                {hasDiffSources ? (
                  <RunDiffSection
                    repoFullName={primaryRepo as string}
                    additionalRepoFullNames={additionalRepos}
                    withRepoPrefix={shouldPrefixDiffs}
                    ref1={baseRef}
                    ref2={headRef}
                    onControlsChange={setDiffControls}
                    classNames={gitDiffViewerClassNames}
                    metadataByRepo={metadataByRepo}
                    heatmapByFile={heatmapByFile}
                    heatmapThreshold={heatmapThreshold}
                  />
                ) : (
                  <div className="flex h-full items-center justify-center p-6 text-sm text-neutral-600 dark:text-neutral-300">
                    Missing repo or branches to show diff.
                  </div>
                )}
              </Suspense>
            </div>
          </div>
        </div>
      </div>
    </FloatingPane>
  );
}
