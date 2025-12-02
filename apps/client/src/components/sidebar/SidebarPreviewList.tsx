import { TaskTree } from "@/components/TaskTree";
import { api } from "@cmux/convex/api";
import type { Id } from "@cmux/convex/dataModel";
import { useQuery as useConvexQuery, useQueries } from "convex/react";
import { useMemo } from "react";

type Props = {
  teamSlugOrId: string;
  limit?: number;
};

const DEFAULT_LIMIT = 10;

export function SidebarPreviewList({
  teamSlugOrId,
  limit = DEFAULT_LIMIT,
}: Props) {
  const previewRuns = useConvexQuery(api.previewRuns.listByTeam, {
    teamSlugOrId,
    limit,
  });

  const list = useMemo(() => previewRuns ?? [], [previewRuns]);

  // Get unique task IDs from preview runs, preserving order (most recent preview run first)
  const taskIds = useMemo(() => {
    const seen = new Set<Id<"tasks">>();
    const ids: Id<"tasks">[] = [];
    for (const run of list) {
      if (run.taskId && !seen.has(run.taskId)) {
        seen.add(run.taskId);
        ids.push(run.taskId);
      }
    }
    return ids;
  }, [list]);

  // Batch fetch all tasks in parallel using useQueries
  const taskQueries = useMemo(() => {
    return taskIds.reduce(
      (acc, taskId) => ({
        ...acc,
        [taskId]: {
          query: api.tasks.getById,
          args: { teamSlugOrId, id: taskId },
        },
      }),
      {} as Record<
        Id<"tasks">,
        {
          query: typeof api.tasks.getById;
          args: { teamSlugOrId: string; id: Id<"tasks"> };
        }
      >
    );
  }, [taskIds, teamSlugOrId]);

  const taskResults = useQueries(
    taskQueries as Parameters<typeof useQueries>[0]
  );

  // Build ordered list of tasks (preserving preview run order)
  const tasks = useMemo(() => {
    return taskIds
      .map((id) => taskResults?.[id])
      .filter((task): task is NonNullable<typeof task> => task != null);
  }, [taskIds, taskResults]);

  if (previewRuns === undefined) {
    return (
      <div className="space-y-px" aria-label="Loading previews">
        {Array.from({ length: 3 }).map((_, index) => (
          <div key={index} className="px-2 py-1.5">
            <div className="h-3 rounded bg-neutral-200 dark:bg-neutral-800 animate-pulse" />
          </div>
        ))}
      </div>
    );
  }

  if (list.length === 0) {
    return (
      <p className="mt-1 pl-2 pr-3 py-2 text-xs text-neutral-500 dark:text-neutral-400 select-none">
        No preview runs
      </p>
    );
  }

  if (tasks.length === 0 && taskIds.length > 0) {
    // Still loading tasks
    return (
      <div className="space-y-px" aria-label="Loading previews">
        {Array.from({ length: Math.min(3, taskIds.length) }).map((_, index) => (
          <div key={index} className="px-2 py-1.5">
            <div className="h-3 rounded bg-neutral-200 dark:bg-neutral-800 animate-pulse" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-px">
      {tasks.map((task) => (
        <TaskTree
          key={task._id}
          task={task}
          defaultExpanded={false}
          teamSlugOrId={teamSlugOrId}
        />
      ))}
    </div>
  );
}

export default SidebarPreviewList;
