import type { AgentConfig } from "../../agentConfig";
import { AWS_BEARER_TOKEN_BEDROCK, AWS_REGION } from "../../apiKeys";
import {
  ANTHROPIC_MODEL_HAIKU_45_ENV,
  ANTHROPIC_MODEL_OPUS_45_ENV,
  ANTHROPIC_MODEL_SONNET_45_ENV,
} from "../../utils/anthropic";
import { checkClaudeRequirements } from "./check-requirements";
import { startClaudeCompletionDetector } from "./completion-detector";
import {
  CLAUDE_KEY_ENV_VARS_TO_UNSET,
  getClaudeEnvironment,
} from "./environment";

/**
 * Apply API keys for Claude agents using AWS Bedrock.
 *
 * Sets up AWS Bedrock bearer token authentication for Claude Code.
 * This is the simpler auth method - just a bearer token, no IAM credentials needed.
 */
const applyClaudeApiKeys: NonNullable<AgentConfig["applyApiKeys"]> = async (
  keys,
) => {
  // Always unset Anthropic-specific env vars to prevent conflicts
  const unsetEnv = [...CLAUDE_KEY_ENV_VARS_TO_UNSET];

  const env: Record<string, string> = {
    // Enable AWS Bedrock mode in Claude Code
    CLAUDE_CODE_USE_BEDROCK: "1",
  };

  // Set AWS Bedrock bearer token if provided
  if (
    keys.AWS_BEARER_TOKEN_BEDROCK &&
    keys.AWS_BEARER_TOKEN_BEDROCK.trim().length > 0
  ) {
    env.AWS_BEARER_TOKEN_BEDROCK = keys.AWS_BEARER_TOKEN_BEDROCK;
  }

  // Set AWS region if provided
  if (keys.AWS_REGION && keys.AWS_REGION.trim().length > 0) {
    env.AWS_REGION = keys.AWS_REGION;
  }

  return {
    env,
    unsetEnv,
  };
};

export const CLAUDE_OPUS_4_5_CONFIG: AgentConfig = {
  name: "claude/opus-4.5",
  command: "bunx",
  args: [
    "@anthropic-ai/claude-code@latest",
    "--model",
    `$${ANTHROPIC_MODEL_OPUS_45_ENV}`,
    "--allow-dangerously-skip-permissions",
    "--dangerously-skip-permissions",
    "--ide",
    "$PROMPT",
  ],
  environment: getClaudeEnvironment,
  checkRequirements: checkClaudeRequirements,
  apiKeys: [AWS_BEARER_TOKEN_BEDROCK, AWS_REGION],
  applyApiKeys: applyClaudeApiKeys,
  completionDetector: startClaudeCompletionDetector,
};

export const CLAUDE_SONNET_4_5_CONFIG: AgentConfig = {
  name: "claude/sonnet-4.5",
  command: "bunx",
  args: [
    "@anthropic-ai/claude-code@latest",
    "--model",
    `$${ANTHROPIC_MODEL_SONNET_45_ENV}`,
    "--allow-dangerously-skip-permissions",
    "--dangerously-skip-permissions",
    "--ide",
    "$PROMPT",
  ],
  environment: getClaudeEnvironment,
  checkRequirements: checkClaudeRequirements,
  apiKeys: [AWS_BEARER_TOKEN_BEDROCK, AWS_REGION],
  applyApiKeys: applyClaudeApiKeys,
  completionDetector: startClaudeCompletionDetector,
};

export const CLAUDE_HAIKU_4_5_CONFIG: AgentConfig = {
  name: "claude/haiku-4.5",
  command: "bunx",
  args: [
    "@anthropic-ai/claude-code@latest",
    "--model",
    `$${ANTHROPIC_MODEL_HAIKU_45_ENV}`,
    "--allow-dangerously-skip-permissions",
    "--dangerously-skip-permissions",
    "--ide",
    "$PROMPT",
  ],
  environment: getClaudeEnvironment,
  checkRequirements: checkClaudeRequirements,
  apiKeys: [AWS_BEARER_TOKEN_BEDROCK, AWS_REGION],
  applyApiKeys: applyClaudeApiKeys,
  completionDetector: startClaudeCompletionDetector,
};
