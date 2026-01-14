import { stackServerAppJs } from "@/lib/utils/stack";
import { createRoute, OpenAPIHono, z } from "@hono/zod-openapi";
import { Octokit } from "octokit";

export const githubBranchesRouter = new OpenAPIHono();

// Schema for branch data
const GithubBranch = z
  .object({
    name: z.string(),
    lastCommitSha: z.string().optional(),
    lastCommitDate: z.string().optional(),
    isDefault: z.boolean().optional(),
  })
  .openapi("GithubBranch");

// --- Default Branch Endpoint (fast - single API call) ---

const DefaultBranchQuery = z
  .object({
    repo: z.string().min(1).openapi({ description: "Repository full name (owner/repo)" }),
  })
  .openapi("GithubDefaultBranchQuery");

const DefaultBranchResponse = z
  .object({
    defaultBranch: z.string().nullable(),
    error: z.string().nullable(),
  })
  .openapi("GithubDefaultBranchResponse");

githubBranchesRouter.openapi(
  createRoute({
    method: "get" as const,
    path: "/integrations/github/default-branch",
    tags: ["Integrations"],
    summary: "Get the default branch for a repository (fast - single API call)",
    request: { query: DefaultBranchQuery },
    responses: {
      200: {
        description: "Default branch response",
        content: {
          "application/json": {
            schema: DefaultBranchResponse,
          },
        },
      },
      401: { description: "Unauthorized" },
    },
  }),
  async (c) => {
    const user = await stackServerAppJs.getUser({ tokenStore: c.req.raw });
    if (!user) {
      return c.text("Unauthorized", 401);
    }

    const { repo } = c.req.valid("query");

    try {
      const githubAccount = await user.getConnectedAccount("github");
      if (!githubAccount) {
        return c.json({ defaultBranch: null, error: "GitHub account not connected" }, 200);
      }

      const { accessToken } = await githubAccount.getAccessToken();
      if (!accessToken || accessToken.trim().length === 0) {
        return c.json({ defaultBranch: null, error: "GitHub access token not found" }, 200);
      }

      const octokit = new Octokit({ auth: accessToken.trim() });
      const [owner, repoName] = repo.split("/");

      const { data } = await octokit.request("GET /repos/{owner}/{repo}", {
        owner: owner!,
        repo: repoName!,
      });

      return c.json({ defaultBranch: data.default_branch, error: null }, 200);
    } catch (error) {
      console.error("[github.branches] Error getting default branch:", error);
      return c.json({
        defaultBranch: null,
        error: error instanceof Error ? error.message : "Failed to get default branch",
      }, 200);
    }
  }
);

// --- Branches List Endpoint (with optional search) ---

const BranchesQuery = z
  .object({
    repo: z.string().min(1).openapi({ description: "Repository full name (owner/repo)" }),
    search: z
      .string()
      .trim()
      .optional()
      .openapi({ description: "Optional search term to filter branches by name" }),
    limit: z.coerce
      .number()
      .min(1)
      .max(100)
      .default(30)
      .optional()
      .openapi({ description: "Max branches to return (default 30, max 100)" }),
    cursor: z
      .string()
      .optional()
      .openapi({ description: "Cursor for pagination (from previous response)" }),
  })
  .openapi("GithubBranchesQuery");

const BranchesResponse = z
  .object({
    branches: z.array(GithubBranch),
    defaultBranch: z.string().nullable(),
    error: z.string().nullable(),
    nextCursor: z.string().nullable(),
    hasMore: z.boolean(),
  })
  .openapi("GithubBranchesResponse");

githubBranchesRouter.openapi(
  createRoute({
    method: "get" as const,
    path: "/integrations/github/branches",
    tags: ["Integrations"],
    summary: "List branches for a repository with optional search filter",
    request: { query: BranchesQuery },
    responses: {
      200: {
        description: "Branches list response",
        content: {
          "application/json": {
            schema: BranchesResponse,
          },
        },
      },
      401: { description: "Unauthorized" },
    },
  }),
  async (c) => {
    const user = await stackServerAppJs.getUser({ tokenStore: c.req.raw });
    if (!user) {
      return c.text("Unauthorized", 401);
    }

    const { repo, search, limit = 30, cursor } = c.req.valid("query");

    try {
      const githubAccount = await user.getConnectedAccount("github");
      if (!githubAccount) {
        return c.json({
          branches: [],
          defaultBranch: null,
          error: "GitHub account not connected",
          nextCursor: null,
          hasMore: false,
        }, 200);
      }

      const { accessToken } = await githubAccount.getAccessToken();
      if (!accessToken || accessToken.trim().length === 0) {
        return c.json({
          branches: [],
          defaultBranch: null,
          error: "GitHub access token not found",
          nextCursor: null,
          hasMore: false,
        }, 200);
      }

      const octokit = new Octokit({ auth: accessToken.trim() });
      const [owner, repoName] = repo.split("/");
      if (!owner || !repoName) {
        return c.json({
          branches: [],
          defaultBranch: null,
          error: "Invalid repository format",
          nextCursor: null,
          hasMore: false,
        }, 200);
      }

      const normalizedSearch = search?.trim().toLowerCase() ?? "";
      const shouldFilter = normalizedSearch.length > 0;
      const pageSize = shouldFilter ? 100 : limit;

      const graphqlResponse = z.object({
        repository: z
          .object({
            defaultBranchRef: z.object({ name: z.string() }).nullable(),
            refs: z.object({
              edges: z.array(
                z.object({
                  cursor: z.string(),
                  node: z.object({
                    name: z.string(),
                    target: z
                      .object({
                        oid: z.string(),
                        committedDate: z.string().optional(),
                      })
                      .passthrough(),
                  }),
                })
              ),
              pageInfo: z.object({
                hasNextPage: z.boolean(),
                endCursor: z.string().nullable(),
              }),
            }),
          })
          .nullable(),
      });

      const query = `
        query($owner: String!, $repo: String!, $first: Int!, $after: String) {
          repository(owner: $owner, name: $repo) {
            defaultBranchRef {
              name
            }
            refs(refPrefix: "refs/heads/", first: $first, after: $after, orderBy: { field: TAG_COMMIT_DATE, direction: DESC }) {
              edges {
                cursor
                node {
                  name
                  target {
                    oid
                    ... on Commit {
                      committedDate
                    }
                  }
                }
              }
              pageInfo {
                hasNextPage
                endCursor
              }
            }
          }
        }
      `;

      const branches: Array<z.infer<typeof GithubBranch>> = [];
      let defaultBranchName: string | null = null;
      let nextCursor: string | null = null;
      let hasMore = false;
      let afterCursor: string | null = cursor ?? null;

      while (branches.length < limit) {
        const rawResponse: unknown = await octokit.graphql(query, {
          owner,
          repo: repoName,
          first: pageSize,
          after: afterCursor ?? null,
        });

        const parsed = graphqlResponse.parse(rawResponse);
        const repoData = parsed.repository;
        if (!repoData) {
          return c.json({
            branches: [],
            defaultBranch: null,
            error: "Repository not found",
            nextCursor: null,
            hasMore: false,
          }, 200);
        }

        if (defaultBranchName === null) {
          defaultBranchName = repoData.defaultBranchRef?.name ?? null;
        }

        const edges = repoData.refs.edges;
        for (let i = 0; i < edges.length; i++) {
          const edge = edges[i];
          const name = edge.node.name;
          if (shouldFilter && !name.toLowerCase().includes(normalizedSearch)) {
            continue;
          }

          branches.push({
            name,
            lastCommitSha: edge.node.target.oid,
            lastCommitDate: edge.node.target.committedDate,
            isDefault: name === defaultBranchName,
          });

          if (branches.length >= limit) {
            nextCursor = edge.cursor;
            hasMore = i < edges.length - 1 || repoData.refs.pageInfo.hasNextPage;
            break;
          }
        }

        if (branches.length >= limit) {
          break;
        }

        if (!repoData.refs.pageInfo.hasNextPage) {
          hasMore = false;
          nextCursor = null;
          break;
        }

        if (!repoData.refs.pageInfo.endCursor) {
          hasMore = false;
          nextCursor = null;
          break;
        }

        afterCursor = repoData.refs.pageInfo.endCursor;
      }

      return c.json({
        branches,
        defaultBranch: defaultBranchName,
        error: null,
        nextCursor,
        hasMore,
      }, 200);
    } catch (error) {
      console.error("[github.branches] Error fetching branches:", error);
      return c.json({
        branches: [],
        defaultBranch: null,
        error: error instanceof Error ? error.message : "Failed to fetch branches",
        nextCursor: null,
        hasMore: false,
      }, 200);
    }
  }
);
