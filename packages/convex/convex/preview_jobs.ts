import { v } from "convex/values";
import { env } from "../_shared/convex-env";
import { internal } from "./_generated/api";
import { internalAction } from "./_generated/server";

export const requestDispatch = internalAction({
  args: {
    previewRunId: v.id("previewRuns"),
  },
  handler: async (ctx, args) => {
    const payload = await ctx.runQuery(internal.previewRuns.getRunWithConfig, {
      previewRunId: args.previewRunId,
    });

    if (!payload?.run || !payload.config) {
      console.warn("[preview-jobs] Missing run/config for dispatch", args);
      return;
    }

    const baseUrl = env.BASE_APP_URL;
    const secret = env.CMUX_TASK_RUN_JWT_SECRET;
    if (!baseUrl || !secret) {
      console.warn("[preview-jobs] BASE_APP_URL or CMUX_TASK_RUN_JWT_SECRET missing");
      return;
    }

    try {
      await ctx.runMutation(internal.previewRuns.markDispatched, {
        previewRunId: args.previewRunId,
      });
    } catch (error) {
      console.error("[preview-jobs] Failed to mark preview run dispatched", {
        previewRunId: args.previewRunId,
        error,
      });
      return;
    }

    try {
      const response = await fetch(`${baseUrl}/api/preview/jobs/dispatch`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${secret}`,
        },
        body: JSON.stringify({
          previewRunId: payload.run._id,
          run: payload.run,
          config: payload.config,
        }),
      });
      if (!response.ok) {
        const text = await response.text();
        console.error("[preview-jobs] Dispatch request failed", {
          previewRunId: args.previewRunId,
          status: response.status,
          body: text.slice(0, 256),
        });
      }
    } catch (error) {
      console.error("[preview-jobs] Failed to invoke preview job dispatcher", {
        previewRunId: args.previewRunId,
        error,
      });
    }
  },
});
