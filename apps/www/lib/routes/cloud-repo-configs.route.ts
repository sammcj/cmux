import { getAccessTokenFromRequest } from "@/lib/utils/auth";
import { getConvex } from "@/lib/utils/get-convex";
import { stackServerAppJs } from "@/lib/utils/stack";
import { verifyTeamAccess } from "@/lib/utils/team-verification";
import { env } from "@/lib/utils/www-env";
import { api } from "@cmux/convex/api";
import { OpenAPIHono, createRoute, z } from "@hono/zod-openapi";
import { HTTPException } from "hono/http-exception";
import { randomBytes } from "node:crypto";

export const cloudRepoConfigsRouter = new OpenAPIHono();

const CloudRepoConfigResponse = z
  .object({
    projectFullName: z.string(),
    maintenanceScript: z.string().optional(),
    envVarsContent: z.string(),
    updatedAt: z.number().optional(),
  })
  .openapi("CloudRepoConfigResponse");

const CloudRepoConfigQuery = z
  .object({
    teamSlugOrId: z.string(),
    projectFullName: z.string(),
  })
  .openapi("CloudRepoConfigQuery");

const CloudRepoConfigBody = z
  .object({
    teamSlugOrId: z.string(),
    projectFullName: z.string(),
    maintenanceScript: z.string().optional(),
    envVarsContent: z.string().default(""),
  })
  .openapi("CloudRepoConfigBody");

async function loadEnvVarsContent(
  dataVaultKey: string | undefined,
): Promise<string> {
  if (!dataVaultKey) return "";
  const store = await stackServerAppJs.getDataVaultStore("cmux-snapshot-envs");
  const value = await store.getValue(dataVaultKey, {
    secret: env.STACK_DATA_VAULT_SECRET,
  });
  return value ?? "";
}

cloudRepoConfigsRouter.openapi(
  createRoute({
    method: "get",
    path: "/cloud-repo-configs",
    summary: "Get cloud repo configuration",
    tags: ["CloudRepoConfigs"],
    request: {
      query: CloudRepoConfigQuery,
    },
    responses: {
      200: {
        description: "Configuration retrieved",
        content: {
          "application/json": {
            schema: CloudRepoConfigResponse.nullable(),
          },
        },
      },
      401: { description: "Unauthorized" },
    },
  }),
  async (c) => {
    const accessToken = await getAccessTokenFromRequest(c.req.raw);
    if (!accessToken) return c.text("Unauthorized", 401);

    const query = c.req.valid("query");

    await verifyTeamAccess({
      req: c.req.raw,
      teamSlugOrId: query.teamSlugOrId,
    });

    const convex = getConvex({ accessToken });
    const config = await convex.query(api.cloudRepoConfigs.get, {
      teamSlugOrId: query.teamSlugOrId,
      projectFullName: query.projectFullName,
    });

    if (!config) {
      return c.json(null);
    }

    const envVarsContent = await loadEnvVarsContent(config.dataVaultKey);

    return c.json({
      projectFullName: config.projectFullName,
      maintenanceScript: config.maintenanceScript ?? undefined,
      envVarsContent,
      updatedAt: config.updatedAt,
    });
  },
);

cloudRepoConfigsRouter.openapi(
  createRoute({
    method: "post",
    path: "/cloud-repo-configs",
    summary: "Create or update cloud repo configuration",
    tags: ["CloudRepoConfigs"],
    request: {
      body: {
        content: {
          "application/json": {
            schema: CloudRepoConfigBody,
          },
        },
        required: true,
      },
    },
    responses: {
      200: {
        description: "Configuration saved",
        content: {
          "application/json": {
            schema: CloudRepoConfigResponse,
          },
        },
      },
      400: { description: "Invalid request" },
      401: { description: "Unauthorized" },
    },
  }),
  async (c) => {
    const accessToken = await getAccessTokenFromRequest(c.req.raw);
    if (!accessToken) return c.text("Unauthorized", 401);

    const body = c.req.valid("json");

    await verifyTeamAccess({
      req: c.req.raw,
      teamSlugOrId: body.teamSlugOrId,
    });

    const convex = getConvex({ accessToken });
    const existing = await convex.query(api.cloudRepoConfigs.get, {
      teamSlugOrId: body.teamSlugOrId,
      projectFullName: body.projectFullName,
    });

    const store = await stackServerAppJs.getDataVaultStore(
      "cmux-snapshot-envs",
    );
    const envVarsContent = body.envVarsContent ?? "";
    let dataVaultKey = existing?.dataVaultKey;
    if (!dataVaultKey) {
      dataVaultKey = `cloud_${randomBytes(16).toString("hex")}`;
    }

    try {
      await store.setValue(dataVaultKey, envVarsContent, {
        secret: env.STACK_DATA_VAULT_SECRET,
      });
    } catch (error) {
      throw new HTTPException(500, {
        message: "Failed to persist environment variables",
        cause: error,
      });
    }

    await convex.mutation(api.cloudRepoConfigs.upsert, {
      teamSlugOrId: body.teamSlugOrId,
      projectFullName: body.projectFullName,
      maintenanceScript: body.maintenanceScript,
      dataVaultKey,
    });

    return c.json({
      projectFullName: body.projectFullName,
      maintenanceScript: body.maintenanceScript,
      envVarsContent,
      updatedAt: Date.now(),
    });
  },
);
