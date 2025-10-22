import { promises as fs } from "node:fs";
import { dirname } from "node:path";

import { log } from "../logger";

const SCREENSHOT_COLLECTOR_LOG_PATH = "/var/log/cmux/screenshot-collector";
const SCREENSHOT_COLLECTOR_DIRECTORY_URL =
  "http://localhost:39378/?folder=/var/log/cmux";

export async function startScreenshotCollection(): Promise<void> {
  const timestamp = new Date().toISOString();
  const message = `${timestamp} start-screenshot-collection triggered\n`;

  try {
    await fs.mkdir(dirname(SCREENSHOT_COLLECTOR_LOG_PATH), {
      recursive: true,
    });
    await fs.appendFile(SCREENSHOT_COLLECTOR_LOG_PATH, message, {
      encoding: "utf8",
    });
    log("INFO", "Screenshot collection trigger recorded", {
      path: SCREENSHOT_COLLECTOR_LOG_PATH,
      openVSCodeUrl: SCREENSHOT_COLLECTOR_DIRECTORY_URL,
    });
  } catch (error) {
    log("ERROR", "Failed to record screenshot collection trigger", {
      path: SCREENSHOT_COLLECTOR_LOG_PATH,
      openVSCodeUrl: SCREENSHOT_COLLECTOR_DIRECTORY_URL,
      error: error instanceof Error ? error.message : String(error),
    });
    throw error;
  }
}
