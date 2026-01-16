/**
 * PostHog analytics for Convex
 * Tracks webhook events for preview.new
 */

const POSTHOG_HOST = "https://us.i.posthog.com";

type PostHogEvent = {
  distinctId: string;
  event: string;
  properties?: Record<string, unknown>;
};

/**
 * Capture a PostHog event from Convex
 * Uses the PostHog capture API directly via fetch
 */
export async function capturePosthogEvent(
  apiKey: string | undefined,
  payload: PostHogEvent
): Promise<void> {
  if (!apiKey) {
    return;
  }

  try {
    await fetch(`${POSTHOG_HOST}/capture/`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        api_key: apiKey,
        distinct_id: payload.distinctId,
        event: payload.event,
        properties: {
          ...payload.properties,
          $lib: "convex",
        },
      }),
    });
  } catch (error) {
    console.error(`[posthog] Failed to capture event "${payload.event}"`, error);
  }
}
