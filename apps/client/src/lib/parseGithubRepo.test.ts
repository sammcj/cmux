import { describe, expect, it } from "vitest";
import {
  parseGithubRepo,
  isValidGithubRepo,
  getGithubRepoFullName,
} from "./parseGithubRepo";

describe("parseGithubRepo", () => {
  it("should parse owner/repo format", () => {
    const result = parseGithubRepo("facebook/react");
    expect(result).toEqual({
      owner: "facebook",
      repo: "react",
      fullName: "facebook/react",
      url: "https://github.com/facebook/react",
    });
  });

  it("should parse https://github.com/owner/repo format", () => {
    const result = parseGithubRepo("https://github.com/vercel/next.js");
    expect(result).toEqual({
      owner: "vercel",
      repo: "next.js",
      fullName: "vercel/next.js",
      url: "https://github.com/vercel/next.js",
    });
  });

  it("should parse https://github.com/owner/repo.git format", () => {
    const result = parseGithubRepo("https://github.com/microsoft/vscode.git");
    expect(result).toEqual({
      owner: "microsoft",
      repo: "vscode",
      fullName: "microsoft/vscode",
      url: "https://github.com/microsoft/vscode",
    });
  });

  it("should parse git@github.com:owner/repo.git format", () => {
    const result = parseGithubRepo("git@github.com:nodejs/node.git");
    expect(result).toEqual({
      owner: "nodejs",
      repo: "node",
      fullName: "nodejs/node",
      url: "https://github.com/nodejs/node",
    });
  });

  it("should handle trailing slash in https URL", () => {
    const result = parseGithubRepo("https://github.com/denoland/deno/");
    expect(result).toEqual({
      owner: "denoland",
      repo: "deno",
      fullName: "denoland/deno",
      url: "https://github.com/denoland/deno",
    });
  });

  it("should handle repos with dashes and underscores", () => {
    const result = parseGithubRepo("my-org/my_awesome-repo");
    expect(result).toEqual({
      owner: "my-org",
      repo: "my_awesome-repo",
      fullName: "my-org/my_awesome-repo",
      url: "https://github.com/my-org/my_awesome-repo",
    });
  });

  it("should handle repos with dots in name", () => {
    const result = parseGithubRepo("vercel/next.js");
    expect(result).toEqual({
      owner: "vercel",
      repo: "next.js",
      fullName: "vercel/next.js",
      url: "https://github.com/vercel/next.js",
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
