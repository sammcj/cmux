#!/usr/bin/env bun
/**
 * Test script for CDP click tracker injection and cursor overlay rendering.
 *
 * This script:
 * 1. Connects to Chrome via CDP
 * 2. Injects a click tracker that captures click coordinates
 * 3. Records the screen while user clicks
 * 4. Writes click events to events.log
 * 5. Runs cursor overlay post-processing
 *
 * Usage:
 *   # Start Chrome with remote debugging:
 *   chromium --remote-debugging-port=9222
 *
 *   # Run this script:
 *   bun run scripts/test-cdp-click-tracker.ts [outputDir]
 */

import { promises as fs } from "node:fs";
import * as path from "node:path";
import { spawn, exec } from "node:child_process";
import { promisify } from "node:util";

const execAsync = promisify(exec);

// Configuration
const DEFAULT_OUTPUT_DIR = "/tmp/test-cdp-click-tracker";
const CDP_PORT = process.env.CDP_PORT || "9222";
const DISPLAY = process.env.DISPLAY || ":0";
const VIDEO_WIDTH = 1920;
const VIDEO_HEIGHT = 1080;
const FRAMERATE = 24;
const RECORD_DURATION_MS = 10000; // 10 seconds

interface CDPTarget {
  type?: string;
  webSocketDebuggerUrl?: string;
  url?: string;
  title?: string;
}

interface ClickEvent {
  timestamp: number;
  event: "click" | "bounding_rect" | "recording_start";
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  screenX?: number;
  screenY?: number;
  target?: string;
}

/**
 * Get CDP WebSocket URL for a page target
 */
async function getCDPWebSocketUrl(): Promise<string | null> {
  try {
    const response = await fetch(`http://127.0.0.1:${CDP_PORT}/json`);
    const targets = (await response.json()) as CDPTarget[];
    const pageTarget = targets.find(
      (t) => t.type === "page" && t.webSocketDebuggerUrl
    );
    return pageTarget?.webSocketDebuggerUrl ?? null;
  } catch (err) {
    console.error(`[CDP] Failed to get targets: ${err}`);
    return null;
  }
}

/**
 * Connect to CDP and inject click tracker
 */
async function injectClickTracker(
  wsUrl: string,
  eventsLogPath: string
): Promise<{ ws: WebSocket; cleanup: () => void }> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl);
    let msgId = 0;
    const clickEvents: ClickEvent[] = [];

    ws.addEventListener("open", () => {
      console.log("[CDP] Connected to browser");

      // Enable Runtime for script injection
      msgId++;
      ws.send(JSON.stringify({ id: msgId, method: "Runtime.enable" }));

      // Enable DOM for node resolution
      msgId++;
      ws.send(JSON.stringify({ id: msgId, method: "DOM.enable" }));

      // Inject click tracker script
      msgId++;
      const trackerScript = `
        (function() {
          if (window.__cmuxClickTracker) return;
          window.__cmuxClickTracker = true;
          window.__cmuxClicks = [];

          document.addEventListener('click', function(e) {
            const rect = e.target.getBoundingClientRect();
            const clickData = {
              timestamp: Date.now(),
              x: rect.x,
              y: rect.y,
              width: rect.width,
              height: rect.height,
              screenX: e.screenX,
              screenY: e.screenY,
              clientX: e.clientX,
              clientY: e.clientY,
              target: e.target.tagName + (e.target.id ? '#' + e.target.id : '') + (e.target.className ? '.' + e.target.className.split(' ')[0] : '')
            };
            window.__cmuxClicks.push(clickData);
            console.log('[CMUX_CLICK]' + JSON.stringify(clickData));
          }, true);

          console.log('[CMUX] Click tracker injected');
        })();
      `;

      ws.send(
        JSON.stringify({
          id: msgId,
          method: "Runtime.evaluate",
          params: { expression: trackerScript },
        })
      );

      // Enable console to capture click events
      msgId++;
      ws.send(JSON.stringify({ id: msgId, method: "Runtime.enable" }));
    });

    ws.addEventListener("message", (event) => {
      try {
        const msg = JSON.parse(String(event.data)) as {
          id?: number;
          method?: string;
          params?: {
            type?: string;
            args?: Array<{ value?: string }>;
          };
        };

        // Capture console messages with click data
        if (msg.method === "Runtime.consoleAPICalled") {
          const args = msg.params?.args;
          if (args && args[0]?.value?.startsWith("[CMUX_CLICK]")) {
            const jsonStr = args[0].value.replace("[CMUX_CLICK]", "");
            try {
              const clickData = JSON.parse(jsonStr) as {
                timestamp: number;
                x: number;
                y: number;
                width: number;
                height: number;
                screenX: number;
                screenY: number;
                target: string;
              };

              // Log bounding_rect event
              const boundingRectEvent: ClickEvent = {
                timestamp: clickData.timestamp,
                event: "bounding_rect",
                x: clickData.x,
                y: clickData.y,
                width: clickData.width,
                height: clickData.height,
              };
              clickEvents.push(boundingRectEvent);

              // Log click event
              const clickEvent: ClickEvent = {
                timestamp: clickData.timestamp + 1,
                event: "click",
                screenX: clickData.screenX,
                screenY: clickData.screenY,
                target: clickData.target,
              };
              clickEvents.push(clickEvent);

              console.log(
                `[CLICK] ${clickData.target} at (${clickData.x + clickData.width / 2}, ${clickData.y + clickData.height / 2})`
              );
            } catch {
              // Ignore parse errors
            }
          }
        }

        if (msg.id === 3) {
          console.log("[CDP] Click tracker injected successfully");
          resolve({
            ws,
            cleanup: async () => {
              // Write all collected events to events.log
              const eventsToWrite = clickEvents.map((e) => JSON.stringify(e)).join("\n");
              if (eventsToWrite) {
                await fs.appendFile(eventsLogPath, eventsToWrite + "\n");
              }
              ws.close();
            },
          });
        }
      } catch {
        // Ignore parse errors
      }
    });

    ws.addEventListener("error", (err) => {
      console.error("[CDP] WebSocket error:", err);
      reject(err);
    });

    setTimeout(() => {
      reject(new Error("CDP connection timeout"));
    }, 5000);
  });
}

/**
 * Start ffmpeg recording
 */
function startRecording(
  outputDir: string
): Promise<{ process: ReturnType<typeof spawn>; stop: () => Promise<void> }> {
  return new Promise((resolve, reject) => {
    const outputPath = path.join(outputDir, "raw.mp4");

    const args = [
      "-y",
      "-f", "x11grab",
      "-draw_mouse", "0",
      "-framerate", String(FRAMERATE),
      "-video_size", `${VIDEO_WIDTH}x${VIDEO_HEIGHT}`,
      "-i", `${DISPLAY}+0,0`,
      "-c:v", "libx264",
      "-preset", "ultrafast",
      "-crf", "26",
      "-pix_fmt", "yuv420p",
      outputPath,
    ];

    console.log(`[FFMPEG] Starting: ffmpeg ${args.join(" ")}`);

    const proc = spawn("ffmpeg", args, {
      env: { ...process.env, DISPLAY },
      stdio: ["pipe", "pipe", "pipe"],
    });

    proc.stderr.on("data", (data) => {
      const msg = String(data);
      if (msg.includes("frame=")) {
        process.stdout.write(".");
      }
    });

    setTimeout(() => {
      if (proc.pid) {
        console.log(`\n[FFMPEG] Recording started (PID ${proc.pid})`);
        resolve({
          process: proc,
          stop: () =>
            new Promise<void>((res) => {
              proc.on("close", () => res());
              proc.kill("SIGINT");
              setTimeout(() => {
                if (!proc.killed) proc.kill("SIGKILL");
                res();
              }, 3000);
            }),
        });
      } else {
        reject(new Error("FFmpeg failed to start"));
      }
    }, 500);

    proc.on("error", reject);
  });
}

/**
 * Run cursor overlay post-processing
 */
async function runPostProcessing(outputDir: string): Promise<void> {
  const script = `
import subprocess, os, sys, json

outdir = "${outputDir}"
events_path = f"{outdir}/events.log"

clicks = []
recording_start_ms = None
last_rect = None

print(f"Reading {events_path}", file=sys.stderr)

try:
    with open(events_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            ev = json.loads(line)
            ts = ev.get("timestamp", 0)
            evt = ev.get("event", "")

            if evt == "recording_start":
                recording_start_ms = ts
                print(f"Recording start: {ts}", file=sys.stderr)
            elif evt == "bounding_rect":
                x, y = ev.get("x", 0), ev.get("y", 0)
                w, h = ev.get("width", 0), ev.get("height", 0)
                cx, cy = int(x + w/2), int(y + h/2 + 85)
                last_rect = (cx, cy, ts)
                print(f"Rect: ({x},{y},{w},{h}) -> ({cx},{cy})", file=sys.stderr)
            elif evt == "click":
                if last_rect:
                    clicks.append((ts, last_rect[0], last_rect[1]))
                    print(f"Click at ({last_rect[0]},{last_rect[1]})", file=sys.stderr)
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)

print(f"Found {len(clicks)} clicks", file=sys.stderr)

if not clicks:
    print("No clicks, just speed up video", file=sys.stderr)
    subprocess.run(f'ffmpeg -y -i "{outdir}/raw.mp4" -vf "setpts=0.25*PTS" "{outdir}/workflow.mp4"', shell=True)
    sys.exit(0)

# Adjust timestamps relative to recording start
if recording_start_ms:
    clicks = [(0.5 + (ts - recording_start_ms) / 1000.0, x, y) for ts, x, y in clicks]
else:
    first_ts = clicks[0][0]
    clicks = [(0.5 + (ts - first_ts) / 1000.0, x, y) for ts, x, y in clicks]

print(f"Adjusted: {clicks}", file=sys.stderr)

# Build cursor overlay filters
filters = []
cx, cy = 960, 540
t0, x0, y0 = clicks[0]
dur = max(t0, 0.1)

yo_x, yo_y = -14, -20
bo_x, bo_y = -6, -12

# Animation from center to first click
yx = f"({cx+yo_x}+({x0+yo_x}-{cx+yo_x})*min(t/{dur},1))"
yy = f"({cy+yo_y}+({y0+yo_y}-{cy+yo_y})*min(t/{dur},1))"
bx = f"({cx+bo_x}+({x0+bo_x}-{cx+bo_x})*min(t/{dur},1))"
by = f"({cy+bo_y}+({y0+bo_y}-{cy+bo_y})*min(t/{dur},1))"

filters.append(f"drawtext=text='●':x='{yx}':y='{yy}':fontsize=36:fontcolor=yellow@0.5:enable='between(t,0,{dur:.2f})'")
filters.append(f"drawtext=text='●':x='{bx}':y='{by}':fontsize=12:fontcolor=black:enable='between(t,0,{dur:.2f})'")

# Static cursor at each click
for i, (t, x, y) in enumerate(clicks):
    end_t = clicks[i+1][0] if i+1 < len(clicks) else 9999
    if end_t <= t:
        continue
    e = f"enable='between(t,{t:.2f},{end_t:.2f})'"
    filters.append(f"drawtext=text='●':x={x+yo_x}:y={y+yo_y}:fontsize=36:fontcolor=yellow@0.5:{e}")
    filters.append(f"drawtext=text='●':x={x+bo_x}:y={y+bo_y}:fontsize=12:fontcolor=black:{e}")

fstr = ",".join(filters)
print(f"Applying {len(filters)} overlay filters", file=sys.stderr)

r = subprocess.run(f'ffmpeg -y -i "{outdir}/raw.mp4" -vf "{fstr}" "{outdir}/with_cursor.mp4"', shell=True)
if r.returncode != 0:
    os.rename(f"{outdir}/raw.mp4", f"{outdir}/workflow.mp4")
    sys.exit(1)

os.remove(f"{outdir}/raw.mp4")

# Get duration and apply speed
probe = subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", f"{outdir}/with_cursor.mp4"], capture_output=True, text=True)
vdur = float(probe.stdout.strip()) if probe.stdout.strip() else 60.0

FAST = 4
segs = []
prev = 0.0
for t, x, y in clicks:
    a_start, a_end = max(0, t - 0.3), min(vdur, t + 0.5)
    if a_start > prev:
        segs.append((prev, a_start, FAST))
    if segs and segs[-1][2] == 1 and a_start <= segs[-1][1]:
        segs[-1] = (segs[-1][0], a_end, 1)
    else:
        segs.append((a_start, a_end, 1))
    prev = a_end
if prev < vdur:
    segs.append((prev, vdur, FAST))

print(f"Speed segments: {segs}", file=sys.stderr)

parts = []
labels = []
for i, (s, e, spd) in enumerate(segs):
    pts = 1.0 / spd
    parts.append(f"[0:v]trim=start={s}:end={e},setpts={pts}*(PTS-STARTPTS)[v{i}]")
    labels.append(f"[v{i}]")

full = ";".join(parts) + ";" + f"{''.join(labels)}concat=n={len(segs)}:v=1:a=0[out]"

sr = subprocess.run(["ffmpeg", "-y", "-i", f"{outdir}/with_cursor.mp4", "-filter_complex", full, "-map", "[out]", f"{outdir}/workflow.mp4"])
if sr.returncode != 0:
    os.rename(f"{outdir}/with_cursor.mp4", f"{outdir}/workflow.mp4")
else:
    os.remove(f"{outdir}/with_cursor.mp4")

print("Done!", file=sys.stderr)
`;

  console.log("[POST] Running cursor overlay post-processing...");
  const { stderr } = await execAsync(`python3 << 'PYSCRIPT'
${script}
PYSCRIPT`);
  if (stderr) console.log(stderr);
}

/**
 * Main
 */
async function main() {
  const outputDir = process.argv[2] || DEFAULT_OUTPUT_DIR;

  console.log("=" .repeat(60));
  console.log("CDP Click Tracker + Cursor Overlay Test");
  console.log("=".repeat(60));
  console.log(`Output: ${outputDir}`);
  console.log(`CDP Port: ${CDP_PORT}`);
  console.log(`Display: ${DISPLAY}`);
  console.log("=".repeat(60));

  await fs.mkdir(outputDir, { recursive: true });

  // Initialize events.log with recording_start
  const eventsLogPath = path.join(outputDir, "events.log");
  const recordingStartTime = Date.now();

  // Check CDP availability
  console.log("\n[1] Connecting to Chrome CDP...");
  const wsUrl = await getCDPWebSocketUrl();
  if (!wsUrl) {
    console.error("ERROR: Cannot connect to Chrome. Start Chrome with:");
    console.error(`  chromium --remote-debugging-port=${CDP_PORT}`);
    process.exit(1);
  }
  console.log(`[CDP] Found browser at ${wsUrl}`);

  // Inject click tracker
  console.log("\n[2] Injecting click tracker...");
  const { cleanup } = await injectClickTracker(wsUrl, eventsLogPath);

  // Write recording_start event
  await fs.writeFile(
    eventsLogPath,
    JSON.stringify({ timestamp: recordingStartTime, event: "recording_start" }) + "\n"
  );

  // Start recording
  console.log("\n[3] Starting screen recording...");
  let recorder: { stop: () => Promise<void> } | null = null;
  try {
    recorder = await startRecording(outputDir);
  } catch (err) {
    console.error(`[FFMPEG] Failed to start: ${err}`);
    console.log("[INFO] Skipping video recording - will test with synthetic data");
  }

  // Wait for clicks
  console.log(`\n[4] Recording for ${RECORD_DURATION_MS / 1000}s - CLICK IN THE BROWSER!`);
  console.log("    Each click will be tracked and logged to events.log");
  await new Promise((r) => setTimeout(r, RECORD_DURATION_MS));

  // Stop recording
  if (recorder) {
    console.log("\n[5] Stopping recording...");
    await recorder.stop();
  }

  // Cleanup CDP connection (writes events to file)
  console.log("\n[6] Writing click events...");
  await cleanup();

  // Check events
  const eventsContent = await fs.readFile(eventsLogPath, "utf-8");
  const eventCount = eventsContent.trim().split("\n").length;
  console.log(`[EVENTS] ${eventCount} events in events.log`);

  // Run post-processing if we have video
  const rawVideoPath = path.join(outputDir, "raw.mp4");
  try {
    await fs.stat(rawVideoPath);
    console.log("\n[7] Running post-processing...");
    await runPostProcessing(outputDir);
  } catch {
    console.log("\n[7] No raw.mp4 found - skipping post-processing");
  }

  // Validate
  console.log("\n" + "=".repeat(60));
  console.log("RESULTS:");
  const files = await fs.readdir(outputDir);
  for (const f of files) {
    const stat = await fs.stat(path.join(outputDir, f));
    console.log(`  ${f}: ${(stat.size / 1024).toFixed(1)} KB`);
  }
  console.log("=".repeat(60));

  // Show events
  console.log("\nEvents captured:");
  console.log(eventsContent);
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
