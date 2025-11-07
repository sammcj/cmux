import { describe, expect, it } from "vitest";
import {
  parseGithubRepo,
  isValidGithubRepo,
  getGithubRepoFullName,
  validateGithubRepoAccess,
} from "./validateGithubRepo";

describe("parseGithubRepo", () => {
  it("should parse owner/repo format", () => {
    const result = parseGithubRepo("facebook/react");
    expect(result).toEqual({
      owner: "facebook",
      repo: "react",
      fullName: "facebook/react",
      url: "https://github.com/facebook/react",
      gitUrl: "https://github.com/facebook/react.git",
    });
  });

  it("should parse https://github.com/owner/repo format", () => {
    const result = parseGithubRepo("https://github.com/vercel/next.js");
    expect(result).toEqual({
      owner: "vercel",
      repo: "next.js",
      fullName: "vercel/next.js",
      url: "https://github.com/vercel/next.js",
      gitUrl: "https://github.com/vercel/next.js.git",
    });
  });

  it("should parse https://github.com/owner/repo.git format", () => {
    const result = parseGithubRepo("https://github.com/microsoft/vscode.git");
    expect(result).toEqual({
      owner: "microsoft",
      repo: "vscode",
      fullName: "microsoft/vscode",
      url: "https://github.com/microsoft/vscode",
      gitUrl: "https://github.com/microsoft/vscode.git",
    });
  });

  it("should parse git@github.com:owner/repo.git format", () => {
    const result = parseGithubRepo("git@github.com:nodejs/node.git");
    expect(result).toEqual({
      owner: "nodejs",
      repo: "node",
      fullName: "nodejs/node",
      url: "https://github.com/nodejs/node",
      gitUrl: "https://github.com/nodejs/node.git",
    });
  });

  it("should handle trailing slash in https URL", () => {
    const result = parseGithubRepo("https://github.com/denoland/deno/");
    expect(result).toEqual({
      owner: "denoland",
      repo: "deno",
      fullName: "denoland/deno",
      url: "https://github.com/denoland/deno",
      gitUrl: "https://github.com/denoland/deno.git",
    });
  });

  it("should handle repos with dashes and underscores", () => {
    const result = parseGithubRepo("my-org/my_awesome-repo");
    expect(result).toEqual({
      owner: "my-org",
      repo: "my_awesome-repo",
      fullName: "my-org/my_awesome-repo",
      url: "https://github.com/my-org/my_awesome-repo",
      gitUrl: "https://github.com/my-org/my_awesome-repo.git",
    });
  });

  it("should handle repos with dots in name", () => {
    const result = parseGithubRepo("vercel/next.js");
    expect(result).toEqual({
      owner: "vercel",
      repo: "next.js",
      fullName: "vercel/next.js",
      url: "https://github.com/vercel/next.js",
      gitUrl: "https://github.com/vercel/next.js.git",
    });
  });

  it("should return null for invalid input", () => {
    expect(parseGithubRepo("")).toBe(null);
    expect(parseGithubRepo("invalid")).toBe(null);
    expect(parseGithubRepo("https://gitlab.com/owner/repo")).toBe(null);
    expect(parseGithubRepo("owner/")).toBe(null);
    expect(parseGithubRepo("/repo")).toBe(null);
  });

  it("should handle case-insensitive github.com", () => {
    const result = parseGithubRepo("HTTPS://GITHUB.COM/owner/repo");
    expect(result).toEqual({
      owner: "owner",
      repo: "repo",
      fullName: "owner/repo",
      url: "https://github.com/owner/repo",
      gitUrl: "https://github.com/owner/repo.git",
    });
  });
});

describe("isValidGithubRepo", () => {
  it("should return true for valid repo URLs", () => {
    expect(isValidGithubRepo("facebook/react")).toBe(true);
    expect(isValidGithubRepo("https://github.com/vercel/next.js")).toBe(true);
    expect(isValidGithubRepo("git@github.com:nodejs/node.git")).toBe(true);
  });

  it("should return false for invalid input", () => {
    expect(isValidGithubRepo("")).toBe(false);
    expect(isValidGithubRepo("invalid")).toBe(false);
    expect(isValidGithubRepo("https://gitlab.com/owner/repo")).toBe(false);
  });
});

describe("getGithubRepoFullName", () => {
  it("should extract full name from various formats", () => {
    expect(getGithubRepoFullName("facebook/react")).toBe("facebook/react");
    expect(getGithubRepoFullName("https://github.com/vercel/next.js")).toBe(
      "vercel/next.js"
    );
    expect(getGithubRepoFullName("git@github.com:nodejs/node.git")).toBe(
      "nodejs/node"
    );
  });

  it("should return null for invalid input", () => {
    expect(getGithubRepoFullName("invalid")).toBe(null);
    expect(getGithubRepoFullName("")).toBe(null);
  });
});

describe("validateGithubRepoAccess", () => {
  it("should return accessible: false for invalid repo format", async () => {
    const mockOctokit = {
      request: async () => {
        throw new Error("Should not be called");
      },
    };

    const result = await validateGithubRepoAccess(mockOctokit, "invalid");
    expect(result.accessible).toBe(false);
    expect(result.error).toBe("Invalid GitHub repository URL or format");
  });

  it("should return accessible: true for valid public repo", async () => {
    const mockOctokit = {
      request: async (route: string, params?: Record<string, unknown>) => {
        expect(route).toBe("GET /repos/{owner}/{repo}");
        expect((params as { owner: string }).owner).toBe("facebook");
        expect((params as { repo: string }).repo).toBe("react");

        return {
          status: 200,
          data: {
            id: 123,
            name: "react",
            full_name: "facebook/react",
            private: false,
            owner: { login: "facebook", type: "Organization" },
            default_branch: "main",
            clone_url: "https://github.com/facebook/react.git",
            ssh_url: "git@github.com:facebook/react.git",
            html_url: "https://github.com/facebook/react",
          },
        };
      },
    };

    const result = await validateGithubRepoAccess(mockOctokit, "facebook/react");
    expect(result.accessible).toBe(true);
    expect(result.repo).toEqual({
      id: 123,
      name: "react",
      fullName: "facebook/react",
      private: false,
      owner: "facebook",
      ownerType: "Organization",
      defaultBranch: "main",
      cloneUrl: "https://github.com/facebook/react.git",
      sshUrl: "git@github.com:facebook/react.git",
      htmlUrl: "https://github.com/facebook/react",
    });
  });

  it("should return accessible: false for 404 not found", async () => {
    const mockOctokit = {
      request: async () => {
        const error = new Error("Not found");
        Object.assign(error, { status: 404 });
        throw error;
      },
    };

    const result = await validateGithubRepoAccess(
      mockOctokit,
      "nonexistent/repo"
    );
    expect(result.accessible).toBe(false);
    expect(result.error).toBe("Repository not found");
  });

  it("should return accessible: false for 403 forbidden", async () => {
    const mockOctokit = {
      request: async () => {
        const error = new Error("Forbidden");
        Object.assign(error, { status: 403 });
        throw error;
      },
    };

    const result = await validateGithubRepoAccess(mockOctokit, "private/repo");
    expect(result.accessible).toBe(false);
    expect(result.error).toBe("Access forbidden - check permissions");
  });

  it("should handle network errors", async () => {
    const mockOctokit = {
      request: async () => {
        throw new Error("Network error");
      },
    };

    const result = await validateGithubRepoAccess(mockOctokit, "owner/repo");
    expect(result.accessible).toBe(false);
    expect(result.error).toBe("Error checking repository access");
  });
});
