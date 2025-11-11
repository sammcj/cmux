import { v } from "convex/values";
import { resolveTeamIdLoose } from "../_shared/team";
import { authMutation, authQuery } from "./users/utils";
import { internalQuery } from "./_generated/server";

const DEFAULT_BROWSER_PROFILE = "chromium" as const;

type BrowserProfile = "chromium" | "firefox" | "webkit";

function normalizeRepoFullName(value: string): string {
  const trimmed = value.trim();
  if (!trimmed.includes("/")) {
    throw new Error("repoFullName must be in the form owner/name");
  }
  return trimmed.replace(/\.git$/i, "").toLowerCase();
}

function normalizeScript(value: string | undefined): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

function normalizeBrowser(profile?: BrowserProfile | null): BrowserProfile {
  if (profile === "firefox" || profile === "webkit") {
    return profile;
  }
  return DEFAULT_BROWSER_PROFILE;
}

export const listByTeam = authQuery({
  args: {
    teamSlugOrId: v.string(),
  },
  handler: async (ctx, args) => {
    const teamId = await resolveTeamIdLoose(ctx, args.teamSlugOrId);
    const configs = await ctx.db
      .query("previewConfigs")
      .withIndex("by_team", (q) => q.eq("teamId", teamId))
      .order("desc")
      .collect();
    return configs;
  },
});

export const get = authQuery({
  args: {
    teamSlugOrId: v.string(),
    previewConfigId: v.id("previewConfigs"),
  },
  handler: async (ctx, args) => {
    const teamId = await resolveTeamIdLoose(ctx, args.teamSlugOrId);
    const config = await ctx.db.get(args.previewConfigId);
    if (!config || config.teamId !== teamId) {
      return null;
    }
    return config;
  },
});

export const getByRepo = authQuery({
  args: {
    teamSlugOrId: v.string(),
    repoFullName: v.string(),
  },
  handler: async (ctx, args) => {
    const teamId = await resolveTeamIdLoose(ctx, args.teamSlugOrId);
    const repoFullName = normalizeRepoFullName(args.repoFullName);
    const config = await ctx.db
      .query("previewConfigs")
      .withIndex("by_team_repo", (q) =>
        q.eq("teamId", teamId).eq("repoFullName", repoFullName),
      )
      .first();
    return config ?? null;
  },
});

export const upsert = authMutation({
  args: {
    teamSlugOrId: v.string(),
    repoFullName: v.string(),
    repoInstallationId: v.optional(v.number()),
    providerConnectionId: v.optional(v.id("providerConnections")),
    repoDefaultBranch: v.optional(v.string()),
    devScript: v.optional(v.string()),
    maintenanceScript: v.optional(v.string()),
    browserProfile: v.optional(
      v.union(
        v.literal("chromium"),
        v.literal("firefox"),
        v.literal("webkit"),
      ),
    ),
    envDataVaultKey: v.optional(v.string()),
    morphSnapshotId: v.optional(v.string()),
    status: v.optional(
      v.union(
        v.literal("active"),
        v.literal("paused"),
        v.literal("disabled"),
      ),
    ),
  },
  handler: async (ctx, args) => {
    const userId = ctx.identity.subject;
    if (!userId) {
      throw new Error("Authentication required");
    }
    const teamId = await resolveTeamIdLoose(ctx, args.teamSlugOrId);
    const repoFullName = normalizeRepoFullName(args.repoFullName);
    const devScript = normalizeScript(args.devScript);
    const maintenanceScript = normalizeScript(args.maintenanceScript);
    const browserProfile = normalizeBrowser(args.browserProfile);
    const now = Date.now();

    const existing = await ctx.db
      .query("previewConfigs")
      .withIndex("by_team_repo", (q) =>
        q.eq("teamId", teamId).eq("repoFullName", repoFullName),
      )
      .first();

    if (existing) {
      await ctx.db.patch(existing._id, {
        repoInstallationId: args.repoInstallationId ?? existing.repoInstallationId,
        providerConnectionId:
          args.providerConnectionId ?? existing.providerConnectionId,
        repoDefaultBranch: args.repoDefaultBranch ?? existing.repoDefaultBranch,
        devScript,
        maintenanceScript,
        browserProfile,
        envDataVaultKey: args.envDataVaultKey ?? existing.envDataVaultKey,
        morphSnapshotId: args.morphSnapshotId ?? existing.morphSnapshotId,
        status: args.status ?? existing.status ?? "active",
        updatedAt: now,
      });
      return existing._id;
    }

    const id = await ctx.db.insert("previewConfigs", {
      teamId,
      createdByUserId: userId,
      repoFullName,
      repoProvider: "github",
      repoInstallationId: args.repoInstallationId,
      providerConnectionId: args.providerConnectionId,
      repoDefaultBranch: args.repoDefaultBranch,
      devScript,
      maintenanceScript,
      browserProfile,
      envDataVaultKey: args.envDataVaultKey,
      morphSnapshotId: args.morphSnapshotId,
      status: args.status ?? "active",
      lastRunAt: undefined,
      createdAt: now,
      updatedAt: now,
    });
    return id;
  },
});

export const updateEnvKey = authMutation({
  args: {
    teamSlugOrId: v.string(),
    previewConfigId: v.id("previewConfigs"),
    envDataVaultKey: v.string(),
  },
  handler: async (ctx, args) => {
    const teamId = await resolveTeamIdLoose(ctx, args.teamSlugOrId);
    const config = await ctx.db.get(args.previewConfigId);
    if (!config || config.teamId !== teamId) {
      throw new Error("Preview configuration not found");
    }
    await ctx.db.patch(config._id, {
      envDataVaultKey: args.envDataVaultKey,
      updatedAt: Date.now(),
    });
  },
});

export const getByTeamAndRepo = internalQuery({
  args: {
    teamId: v.string(),
    repoFullName: v.string(),
  },
  handler: async (ctx, args) => {
    const repoFullName = normalizeRepoFullName(args.repoFullName);
    const config = await ctx.db
      .query("previewConfigs")
      .withIndex("by_team_repo", (q) =>
        q.eq("teamId", args.teamId).eq("repoFullName", repoFullName),
      )
      .first();
    return config ?? null;
  },
});
