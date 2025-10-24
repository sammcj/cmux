#!/usr/bin/env bun

import { execFile } from "node:child_process";
import { randomUUID } from "node:crypto";
import process from "node:process";
import readline from "node:readline/promises";
import { promisify } from "node:util";
import {
  startAutomatedPrReview,
  type PrReviewJobContext,
} from "../src/pr-review";
import { parsePrUrl } from "./pr-review/github";

const DEFAULT_PR_URL = "https://github.com/manaflow-ai/cmux/pull/653";
const execFileAsync = promisify(execFile);

interface CliOptions {
  prUrl: string | null;
  isProduction: boolean;
  showDiffLineNumbers: boolean | null;
  showContextLineNumbers: boolean | null;
}

function parseCliArgs(argv: readonly string[]): CliOptions {
  const remainingArgs: string[] = [];
  let isProduction = false;
  let showDiffLineNumbers: boolean | null = null;
  let showContextLineNumbers: boolean | null = null;

  argv.forEach((arg) => {
    if (arg === "--production") {
      isProduction = true;
      return;
    }
    if (arg === "--diff-line-numbers") {
      showDiffLineNumbers = true;
      return;
    }
    if (arg === "--no-diff-line-numbers") {
      showDiffLineNumbers = false;
      return;
    }
    if (arg === "--diff-context-line-numbers") {
      showContextLineNumbers = true;
      return;
    }
    if (arg === "--no-diff-context-line-numbers") {
      showContextLineNumbers = false;
      return;
    }
    remainingArgs.push(arg);
  });

  return {
    prUrl: remainingArgs[0] ?? null,
    isProduction,
    showDiffLineNumbers,
    showContextLineNumbers,
  };
}

async function resolveCommitRef(
  repoFullName: string,
  prNumber: number
): Promise<string> {
  const prIdentifier = `${repoFullName}#${prNumber}`;
  try {
    const { stdout } = await execFileAsync("gh", [
      "pr",
      "view",
      String(prNumber),
      "--repo",
      repoFullName,
      "--json",
      "headRefOid",
      "--jq",
      ".headRefOid",
    ]);
    const commitRef = stdout.trim();
    if (!commitRef) {
      throw new Error(
        `GitHub CLI returned an empty commit ref for ${prIdentifier}`
      );
    }
    return commitRef;
  } catch (error) {
    const baseMessage = `Failed to fetch head commit for ${prIdentifier} via gh`;
    if (error instanceof Error) {
      throw new Error(`${baseMessage}: ${error.message}`, { cause: error });
    }
    throw new Error(baseMessage);
  }
}

async function main(): Promise<void> {
  const {
    prUrl: prUrlArg,
    isProduction,
    showDiffLineNumbers,
    showContextLineNumbers,
  } = parseCliArgs(
    process.argv.slice(2)
  );
  if (isProduction) {
    console.log("[cli] Production mode enabled via --production flag.");
  }

  const productionMode =
    isProduction ||
    process.env.NODE_ENV === "production" ||
    process.env.CMUX_PR_REVIEW_ENV === "production";

  const prUrlInput = prUrlArg ?? DEFAULT_PR_URL;
  const prUrl = prUrlInput.trim();
  if (prUrl.length === 0) {
    throw new Error("PR URL argument cannot be empty");
  }

  const parsed = parsePrUrl(prUrl);
  const repoFullName = `${parsed.owner}/${parsed.repo}`;
  const repoUrl = `https://github.com/${repoFullName}.git`;
  const jobId = randomUUID();
  const sandboxLabel = randomUUID();

  console.log(`[cli] Starting PR review for ${repoFullName}#${parsed.number}`);
  const commitRef = await resolveCommitRef(repoFullName, parsed.number);

  const config: PrReviewJobContext = {
    jobId,
    teamId: "780c4397-90dd-47f1-b336-b8c376039db5",
    repoFullName,
    repoUrl,
    prNumber: parsed.number,
    prUrl,
    commitRef,
    morphSnapshotId: "snapshot_vb7uqz8o",
    productionMode,
  };

  if (showDiffLineNumbers !== null) {
    config.showDiffLineNumbers = showDiffLineNumbers;
  }
  if (showContextLineNumbers !== null) {
    config.showContextLineNumbers = showContextLineNumbers;
  }

  try {
    await startAutomatedPrReview(config);
    console.log(
      `[cli] Review launched (jobId=${jobId}, sandboxHint=${sandboxLabel}).`
    );
    console.log("[cli] Press Enter to exit.");
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    await rl.question("");
    rl.close();
  } catch (error) {
    const message =
      error instanceof Error ? error.message : String(error ?? "unknown error");
    console.error(`[cli] Review setup failed: ${message}`);
    throw error;
  }
}

await main().catch((error) => {
  console.error(
    error instanceof Error ? (error.stack ?? error.message) : error
  );
  process.exit(1);
});
