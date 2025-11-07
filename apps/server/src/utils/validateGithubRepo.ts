/**
 * Server-side GitHub repository validation and parsing
 */

export interface ParsedGithubRepo {
  owner: string;
  repo: string;
  fullName: string;
  url: string;
  gitUrl: string;
}

/**
 * Parse a GitHub repository URL or owner/repo string
 * Supports various formats:
 * - https://github.com/owner/repo
 * - https://github.com/owner/repo.git
 * - git@github.com:owner/repo.git
 * - owner/repo
 */
export function parseGithubRepo(input: string): ParsedGithubRepo | null {
  if (!input || typeof input !== "string") {
    return null;
  }

  const trimmed = input.trim();

  // Pattern 1: owner/repo (simple format)
  const simpleMatch = trimmed.match(/^([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_.-]+)$/);
  if (simpleMatch) {
    const [, owner, repo] = simpleMatch;
    const cleanRepo = repo.replace(/\.git$/, "");
    return {
      owner,
      repo: cleanRepo,
      fullName: `${owner}/${cleanRepo}`,
      url: `https://github.com/${owner}/${cleanRepo}`,
      gitUrl: `https://github.com/${owner}/${cleanRepo}.git`,
    };
  }

  // Pattern 2: https://github.com/owner/repo or https://github.com/owner/repo.git
  const httpsMatch = trimmed.match(
    /^https?:\/\/github\.com\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_.-]+?)(?:\.git)?(?:\/)?$/i
  );
  if (httpsMatch) {
    const [, owner, repo] = httpsMatch;
    const cleanRepo = repo.replace(/\.git$/, "");
    return {
      owner,
      repo: cleanRepo,
      fullName: `${owner}/${cleanRepo}`,
      url: `https://github.com/${owner}/${cleanRepo}`,
      gitUrl: `https://github.com/${owner}/${cleanRepo}.git`,
    };
  }

  // Pattern 3: git@github.com:owner/repo.git
  const sshMatch = trimmed.match(
    /^git@github\.com:([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_.-]+?)(?:\.git)?$/i
  );
  if (sshMatch) {
    const [, owner, repo] = sshMatch;
    const cleanRepo = repo.replace(/\.git$/, "");
    return {
      owner,
      repo: cleanRepo,
      fullName: `${owner}/${cleanRepo}`,
      url: `https://github.com/${owner}/${cleanRepo}`,
      gitUrl: `https://github.com/${owner}/${cleanRepo}.git`,
    };
  }

  return null;
}

/**
 * Validate if a string is a valid GitHub repository URL or identifier
 */
export function isValidGithubRepo(input: string): boolean {
  return parseGithubRepo(input) !== null;
}

/**
 * Get the full repository name (owner/repo) from a URL
 */
export function getGithubRepoFullName(input: string): string | null {
  const parsed = parseGithubRepo(input);
  return parsed ? parsed.fullName : null;
}

/**
 * Validate GitHub repository access using Octokit
 * Returns repo information if accessible, null if not found or no access
 */
export async function validateGithubRepoAccess(
  octokit: {
    request: (route: string, params?: Record<string, unknown>) => Promise<{
      status: number;
      data: {
        id: number;
        name: string;
        full_name: string;
        private: boolean;
        owner: { login: string; type: string };
        default_branch: string;
        clone_url: string;
        ssh_url: string;
        html_url: string;
      };
    }>;
  },
  repoInput: string
): Promise<{
  accessible: boolean;
  repo?: {
    id: number;
    name: string;
    fullName: string;
    private: boolean;
    owner: string;
    ownerType: string;
    defaultBranch: string;
    cloneUrl: string;
    sshUrl: string;
    htmlUrl: string;
  };
  error?: string;
}> {
  const parsed = parseGithubRepo(repoInput);

  if (!parsed) {
    return {
      accessible: false,
      error: "Invalid GitHub repository URL or format",
    };
  }

  try {
    const response = await octokit.request("GET /repos/{owner}/{repo}", {
      owner: parsed.owner,
      repo: parsed.repo,
    });

    if (response.status === 200) {
      return {
        accessible: true,
        repo: {
          id: response.data.id,
          name: response.data.name,
          fullName: response.data.full_name,
          private: response.data.private,
          owner: response.data.owner.login,
          ownerType: response.data.owner.type,
          defaultBranch: response.data.default_branch,
          cloneUrl: response.data.clone_url,
          sshUrl: response.data.ssh_url,
          htmlUrl: response.data.html_url,
        },
      };
    }

    return {
      accessible: false,
      error: "Repository not accessible",
    };
  } catch (error) {
    if (error && typeof error === "object" && "status" in error) {
      if (error.status === 404) {
        return {
          accessible: false,
          error: "Repository not found",
        };
      }
      if (error.status === 403) {
        return {
          accessible: false,
          error: "Access forbidden - check permissions",
        };
      }
    }

    console.error("Error validating GitHub repo access:", error);
    return {
      accessible: false,
      error: "Error checking repository access",
    };
  }
}
