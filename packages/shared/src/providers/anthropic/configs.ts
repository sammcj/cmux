import type { AgentConfig } from "../../agentConfig";
import { ANTHROPIC_API_KEY, CLAUDE_CODE_OAUTH_TOKEN } from "../../apiKeys";
import { checkClaudeRequirements } from "./check-requirements";
import { startClaudeCompletionDetector } from "./completion-detector";
import {
  CLAUDE_KEY_ENV_VARS_TO_UNSET,
  getClaudeEnvironment,
} from "./environment";

// Flag to enable user-provided API keys (OAuth token and ANTHROPIC_API_KEY)
// When false, always use platform-provided AWS Bedrock credentials
// Set to false for testing Bedrock integration
const ENABLE_USER_API_KEYS = false;

// Bedrock model IDs from environment variables (required at runtime, not load time)
// These are read lazily to avoid breaking Convex module analysis
function getBedrockModelId(envVar: string): string {
  const value = process.env[envVar];
  if (!value) {
    throw new Error(`Missing required environment variable: ${envVar}`);
  }
  return value;
}

/**
 * Create applyApiKeys function for a specific Bedrock model.
 *
 * Priority order (when ENABLE_USER_API_KEYS is true):
 * 1. CLAUDE_CODE_OAUTH_TOKEN - user's OAuth token (user pays via subscription)
 * 2. ANTHROPIC_API_KEY - user's own API key
 * 3. AWS Bedrock - platform-provided credentials (fallback)
 *
 * When ENABLE_USER_API_KEYS is false:
 * - Always use AWS Bedrock with platform-provided credentials
 *
 * Note: Claude Code with Bedrock requires the model to be set via the ANTHROPIC_MODEL
 * environment variable (not via --model CLI flag).
 */
function createApplyClaudeApiKeys(
  bedrockModelEnvVar: string,
): NonNullable<AgentConfig["applyApiKeys"]> {
  return async (keys) => {
    // Read model ID lazily at runtime (not at module load time)
    const bedrockModelId = getBedrockModelId(bedrockModelEnvVar);
    // Base env vars to unset (prevent conflicts)
    const unsetEnv = [...CLAUDE_KEY_ENV_VARS_TO_UNSET];

    if (ENABLE_USER_API_KEYS) {
      // Priority 1: OAuth token (user pays via their subscription)
      const oauthToken = keys.CLAUDE_CODE_OAUTH_TOKEN;
      if (oauthToken && oauthToken.trim().length > 0) {
        return {
          env: {
            CLAUDE_CODE_OAUTH_TOKEN: oauthToken,
          },
          unsetEnv,
        };
      }

      // Priority 2: User's own ANTHROPIC_API_KEY
      const userApiKey = keys.ANTHROPIC_API_KEY;
      if (userApiKey && userApiKey.trim().length > 0) {
        return {
          env: {
            ANTHROPIC_API_KEY: userApiKey,
          },
          // Don't unset ANTHROPIC_API_KEY since we're using it
          unsetEnv: unsetEnv.filter((v) => v !== "ANTHROPIC_API_KEY"),
        };
      }
    }

    // Priority 3 (or only option when ENABLE_USER_API_KEYS is false):
    // AWS Bedrock with platform-provided credentials
    const env: Record<string, string> = {
      // Enable AWS Bedrock mode in Claude Code
      CLAUDE_CODE_USE_BEDROCK: "1",
      // Claude Code requires ANTHROPIC_MODEL env var for Bedrock
      ANTHROPIC_MODEL: bedrockModelId,
    };

    // AWS Bedrock credentials are injected by agentSpawner from server environment
    if (keys.AWS_BEARER_TOKEN_BEDROCK) {
      env.AWS_BEARER_TOKEN_BEDROCK = keys.AWS_BEARER_TOKEN_BEDROCK;
    }
    if (keys.AWS_REGION) {
      env.AWS_REGION = keys.AWS_REGION;
    }

    return {
      env,
      unsetEnv,
    };
  };
}

// API keys shown in UI for user configuration
// Only show user API key options if enabled
const claudeApiKeys = ENABLE_USER_API_KEYS
  ? [CLAUDE_CODE_OAUTH_TOKEN, ANTHROPIC_API_KEY]
  : [];

export const CLAUDE_OPUS_4_5_CONFIG: AgentConfig = {
  name: "claude/opus-4.5",
  command: "bunx",
  args: [
    "@anthropic-ai/claude-code@latest",
    "--allow-dangerously-skip-permissions",
    "--dangerously-skip-permissions",
    "--ide",
    "$PROMPT",
  ],
  environment: getClaudeEnvironment,
  checkRequirements: checkClaudeRequirements,
  apiKeys: claudeApiKeys,
  applyApiKeys: createApplyClaudeApiKeys("ANTHROPIC_MODEL_OPUS_45"),
  completionDetector: startClaudeCompletionDetector,
};

export const CLAUDE_SONNET_4_5_CONFIG: AgentConfig = {
  name: "claude/sonnet-4.5",
  command: "bunx",
  args: [
    "@anthropic-ai/claude-code@latest",
    "--allow-dangerously-skip-permissions",
    "--dangerously-skip-permissions",
    "--ide",
    "$PROMPT",
  ],
  environment: getClaudeEnvironment,
  checkRequirements: checkClaudeRequirements,
  apiKeys: claudeApiKeys,
  applyApiKeys: createApplyClaudeApiKeys("ANTHROPIC_MODEL_SONNET_45"),
  completionDetector: startClaudeCompletionDetector,
};

export const CLAUDE_HAIKU_4_5_CONFIG: AgentConfig = {
  name: "claude/haiku-4.5",
  command: "bunx",
  args: [
    "@anthropic-ai/claude-code@latest",
    "--allow-dangerously-skip-permissions",
    "--dangerously-skip-permissions",
    "--ide",
    "$PROMPT",
  ],
  environment: getClaudeEnvironment,
  checkRequirements: checkClaudeRequirements,
  apiKeys: claudeApiKeys,
  applyApiKeys: createApplyClaudeApiKeys("ANTHROPIC_MODEL_HAIKU_45"),
  completionDetector: startClaudeCompletionDetector,
};
