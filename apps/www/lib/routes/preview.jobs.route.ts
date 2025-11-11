import { waitUntil } from "@vercel/functions";
import { z } from "zod";
import { OpenAPIHono, createRoute } from "@hono/zod-openapi";
import { env } from "@/lib/utils/www-env";

export const previewJobsRouter = new OpenAPIHono();

const DispatchBody = z
  .object({
    previewRunId: z.string(),
    run: z.record(z.string(), z.unknown()),
    config: z.record(z.string(), z.unknown()),
  })
  .openapi("PreviewJobDispatch");

function isAuthorized(headerValue: string | null): boolean {
  if (!headerValue) {
    return false;
  }
  const [scheme, token] = headerValue.split(" ");
  if (scheme !== "Bearer") {
    return false;
  }
  return token === env.CMUX_TASK_RUN_JWT_SECRET;
}

async function markRunFailed(previewRunId: string, reason: string) {
  try {
    console.warn("[preview-jobs] Preview dispatcher stub", {
      previewRunId,
      reason,
    });
  } catch (error) {
    console.error("[preview-jobs] Failed to record preview result", {
      previewRunId,
      error,
    });
  }
}

previewJobsRouter.openapi(
  createRoute({
    method: "post" as const,
    path: "/preview/jobs/dispatch",
    tags: ["Preview"],
    request: {
      body: {
        content: {
          "application/json": {
            schema: DispatchBody,
          },
        },
        required: true,
      },
    },
    responses: {
      202: { description: "Job accepted" },
      401: { description: "Unauthorized" },
    },
  }),
  async (c) => {
    if (!isAuthorized(c.req.header("authorization") ?? null)) {
      return c.text("Unauthorized", 401);
    }

    const body = c.req.valid("json");
    waitUntil(
      markRunFailed(body.previewRunId, "Preview job execution not implemented yet"),
    );
    return c.text("accepted", 202);
  },
);
