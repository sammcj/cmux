/**
 * Parse and validate GitHub repository URLs
 */

export interface ParsedGithubRepo {
  owner: string;
  repo: string;
  fullName: string;
  url: string;
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
    return {
      owner,
      repo: repo.replace(/\.git$/, ""),
      fullName: `${owner}/${repo.replace(/\.git$/, "")}`,
      url: `https://github.com/${owner}/${repo.replace(/\.git$/, "")}`,
    };
  }

  // Pattern 2: https://github.com/owner/repo or https://github.com/owner/repo.git
  const httpsMatch = trimmed.match(
    /^https?:\/\/github\.com\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_.-]+?)(?:\.git)?(?:\/)?$/i
  );
  if (httpsMatch) {
    const [, owner, repo] = httpsMatch;
    return {
      owner,
      repo: repo.replace(/\.git$/, ""),
      fullName: `${owner}/${repo.replace(/\.git$/, "")}`,
      url: `https://github.com/${owner}/${repo.replace(/\.git$/, "")}`,
    };
  }

  // Pattern 3: git@github.com:owner/repo.git
  const sshMatch = trimmed.match(
    /^git@github\.com:([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_.-]+?)(?:\.git)?$/i
  );
  if (sshMatch) {
    const [, owner, repo] = sshMatch;
    return {
      owner,
      repo: repo.replace(/\.git$/, ""),
      fullName: `${owner}/${repo.replace(/\.git$/, "")}`,
      url: `https://github.com/${owner}/${repo.replace(/\.git$/, "")}`,
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
