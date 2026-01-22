#!/usr/bin/env bun
/**
 * Test script for CDP bounding box tracking with video recording and cursor overlay.
 *
 * This tests the full video pipeline:
 * 1. Connects to Chrome via CDP
 * 2. Starts video recording with ffmpeg x11grab
 * 3. Clicks elements and tracks bounding boxes via CDP
 * 4. Logs click events to events.log
 * 5. Stops recording and runs cursor overlay post-processing
 *
 * Usage:
 *   # Start Chrome with remote debugging:
 *   chromium --remote-debugging-port=9222
 *
 *   # Run test:
 *   bun run scripts/test-cdp-bounding-box.ts
 */

import { promises as fs } from "node:fs";
import * as path from "node:path";
import * as http from "node:http";
import { spawn } from "node:child_process";
import { exec } from "node:child_process";
import { promisify } from "node:util";

const execAsync = promisify(exec);

// Configuration
const CDP_PORT = process.env.CDP_PORT || "9222";
const TEST_OUTPUT_DIR = "/tmp/test-cdp-bounding-box";
const HTTP_PORT = 19998;
const VIDEO_WIDTH = 1920;
const VIDEO_HEIGHT = 1080;
const FRAMERATE = 24;
const BROWSER_CHROME_OFFSET = 85; // Browser chrome/toolbar height

interface CDPTarget {
  id: string;
  type: string;
  title: string;
  url: string;
  webSocketDebuggerUrl?: string;
}

interface BoundingRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

interface TestElement {
  id: string;
  expectedX: number;
  expectedY: number;
  expectedWidth: number;
  expectedHeight: number;
}

// Test page with elements at known positions
const TEST_PAGE_HTML = `<!DOCTYPE html>
<html>
<head>
  <title>CDP Bounding Box Test Page</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      width: 1920px;
      height: 1080px;
      background: #1a1a2e;
      font-family: monospace;
      position: relative;
    }
    .test-element {
      position: absolute;
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-size: 14px;
      border: 2px solid white;
      cursor: pointer;
      transition: transform 0.1s, background 0.1s;
    }
    .test-element:hover {
      transform: scale(1.05);
    }
    .test-element:active {
      transform: scale(0.95);
    }
    .test-element.clicked {
      background: #00ff00 !important;
      animation: pulse 0.3s ease-out;
    }
    @keyframes pulse {
      0% { transform: scale(1); }
      50% { transform: scale(1.1); }
      100% { transform: scale(1); }
    }
    #btn1 { left: 100px; top: 100px; width: 200px; height: 50px; background: #e94560; }
    #btn2 { left: 500px; top: 200px; width: 150px; height: 60px; background: #0f3460; }
    #btn3 { left: 900px; top: 150px; width: 180px; height: 45px; background: #16213e; }
    #btn4 { left: 200px; top: 400px; width: 250px; height: 80px; background: #533483; }
    #btn5 { left: 700px; top: 500px; width: 300px; height: 100px; background: #e94560; }
    #btn6 { left: 1200px; top: 300px; width: 120px; height: 40px; background: #0f3460; }
    #btn7 { left: 1400px; top: 600px; width: 200px; height: 70px; background: #16213e; }
    #btn8 { left: 50px; top: 700px; width: 400px; height: 120px; background: #533483; }
    .info {
      position: fixed;
      bottom: 10px;
      right: 10px;
      background: rgba(0,0,0,0.8);
      color: #0f0;
      padding: 10px;
      font-size: 12px;
      max-width: 400px;
    }
    .click-indicator {
      position: absolute;
      width: 20px;
      height: 20px;
      border-radius: 50%;
      background: rgba(255, 255, 0, 0.8);
      pointer-events: none;
      animation: click-ripple 0.5s ease-out forwards;
    }
    @keyframes click-ripple {
      0% { transform: scale(0); opacity: 1; }
      100% { transform: scale(3); opacity: 0; }
    }
  </style>
</head>
<body>
  <div id="btn1" class="test-element">btn1 (100,100) 200x50</div>
  <div id="btn2" class="test-element">btn2 (500,200) 150x60</div>
  <div id="btn3" class="test-element">btn3 (900,150) 180x45</div>
  <div id="btn4" class="test-element">btn4 (200,400) 250x80</div>
  <div id="btn5" class="test-element">btn5 (700,500) 300x100</div>
  <div id="btn6" class="test-element">btn6 (1200,300) 120x40</div>
  <div id="btn7" class="test-element">btn7 (1400,600) 200x70</div>
  <div id="btn8" class="test-element">btn8 (50,700) 400x120</div>
  <div class="info" id="info">Click the buttons to test...</div>
  <script>
    // Visual feedback for clicks
    document.addEventListener('click', (e) => {
      // Create click indicator
      const indicator = document.createElement('div');
      indicator.className = 'click-indicator';
      indicator.style.left = (e.clientX - 10) + 'px';
      indicator.style.top = (e.clientY - 10) + 'px';
      document.body.appendChild(indicator);
      setTimeout(() => indicator.remove(), 500);

      const el = e.target;
      if (el.classList.contains('test-element')) {
        el.classList.add('clicked');
        setTimeout(() => el.classList.remove('clicked'), 300);
        const rect = el.getBoundingClientRect();
        document.getElementById('info').innerHTML =
          'Clicked: ' + el.id + '<br>' +
          'Position: (' + e.clientX + ', ' + e.clientY + ')<br>' +
          'Element rect: ' + JSON.stringify({
            x: rect.x, y: rect.y, width: rect.width, height: rect.height
          });
      }
    });
  </script>
</body>
</html>`;

// Expected positions for test elements
const TEST_ELEMENTS: TestElement[] = [
  { id: "btn1", expectedX: 100, expectedY: 100, expectedWidth: 200, expectedHeight: 50 },
  { id: "btn2", expectedX: 500, expectedY: 200, expectedWidth: 150, expectedHeight: 60 },
  { id: "btn3", expectedX: 900, expectedY: 150, expectedWidth: 180, expectedHeight: 45 },
  { id: "btn4", expectedX: 200, expectedY: 400, expectedWidth: 250, expectedHeight: 80 },
  { id: "btn5", expectedX: 700, expectedY: 500, expectedWidth: 300, expectedHeight: 100 },
  { id: "btn6", expectedX: 1200, expectedY: 300, expectedWidth: 120, expectedHeight: 40 },
  { id: "btn7", expectedX: 1400, expectedY: 600, expectedWidth: 200, expectedHeight: 70 },
  { id: "btn8", expectedX: 50, expectedY: 700, expectedWidth: 400, expectedHeight: 120 },
];

/**
 * Start HTTP server for test page
 */
function startTestServer(): Promise<http.Server> {
  return new Promise((resolve) => {
    const server = http.createServer((req, res) => {
      res.writeHead(200, { "Content-Type": "text/html" });
      res.end(TEST_PAGE_HTML);
    });
    server.listen(HTTP_PORT, () => {
      console.log(`[HTTP] Test page at http://localhost:${HTTP_PORT}`);
      resolve(server);
    });
  });
}

/**
 * Get CDP WebSocket URL
 */
async function getCDPTarget(): Promise<CDPTarget | null> {
  try {
    const response = await fetch(`http://127.0.0.1:${CDP_PORT}/json`);
    const targets = (await response.json()) as CDPTarget[];
    return targets.find((t) => t.type === "page" && t.webSocketDebuggerUrl) ?? null;
  } catch (err) {
    console.error(`[CDP] Failed to connect: ${err}`);
    return null;
  }
}

/**
 * CDP WebSocket wrapper for sending commands
 */
class CDPSession {
  private ws: WebSocket;
  private msgId = 0;
  private pendingRequests = new Map<number, { resolve: (v: unknown) => void; reject: (e: Error) => void }>();

  constructor(ws: WebSocket) {
    this.ws = ws;
    ws.addEventListener("message", (event) => {
      const msg = JSON.parse(String(event.data)) as { id?: number; result?: unknown; error?: { message: string } };
      if (msg.id !== undefined) {
        const pending = this.pendingRequests.get(msg.id);
        if (pending) {
          this.pendingRequests.delete(msg.id);
          if (msg.error) {
            pending.reject(new Error(msg.error.message));
          } else {
            pending.resolve(msg.result);
          }
        }
      }
    });
  }

  async send(method: string, params?: Record<string, unknown>): Promise<unknown> {
    return new Promise((resolve, reject) => {
      this.msgId++;
      const id = this.msgId;
      this.pendingRequests.set(id, { resolve, reject });
      this.ws.send(JSON.stringify({ id, method, params }));
      setTimeout(() => {
        if (this.pendingRequests.has(id)) {
          this.pendingRequests.delete(id);
          reject(new Error(`Timeout waiting for ${method}`));
        }
      }, 5000);
    });
  }

  close() {
    this.ws.close();
  }
}

/**
 * Connect to CDP and return session
 */
function connectCDP(wsUrl: string): Promise<CDPSession> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl);
    ws.addEventListener("open", () => resolve(new CDPSession(ws)));
    ws.addEventListener("error", (e) => reject(e));
    setTimeout(() => reject(new Error("CDP connection timeout")), 5000);
  });
}

/**
 * Get backendNodeId for an element by selector
 */
async function getBackendNodeId(cdp: CDPSession, selector: string): Promise<number | null> {
  // Get document
  const doc = (await cdp.send("DOM.getDocument")) as { root: { nodeId: number } };

  // Query selector
  const result = (await cdp.send("DOM.querySelector", {
    nodeId: doc.root.nodeId,
    selector,
  })) as { nodeId: number };

  if (!result.nodeId) return null;

  // Get backendNodeId
  const described = (await cdp.send("DOM.describeNode", {
    nodeId: result.nodeId,
  })) as { node: { backendNodeId: number } };

  return described.node.backendNodeId;
}

/**
 * Get bounding rect via CDP - this mirrors the logic in index.ts getBoundingRectFromCDP
 */
async function getBoundingRectFromCDP(
  cdp: CDPSession,
  uid: string
): Promise<BoundingRect | null> {
  // Parse uid format: "contextId_backendNodeId"
  const parts = uid.split("_");
  const backendNodeIdStr = parts[parts.length - 1] ?? "";
  const backendNodeId = parseInt(backendNodeIdStr, 10);

  if (isNaN(backendNodeId)) {
    console.error(`[CDP] Invalid uid format: ${uid}`);
    return null;
  }

  try {
    // Step 1: Resolve backendNodeId to RemoteObject
    const resolved = (await cdp.send("DOM.resolveNode", {
      backendNodeId,
    })) as { object: { objectId: string } };

    if (!resolved.object?.objectId) {
      console.error(`[CDP] Could not resolve backendNodeId ${backendNodeId}`);
      return null;
    }

    // Step 2: Call getBoundingClientRect on the element
    const result = (await cdp.send("Runtime.callFunctionOn", {
      objectId: resolved.object.objectId,
      functionDeclaration:
        "function() { const r = this.getBoundingClientRect(); return JSON.stringify({x:r.x,y:r.y,width:r.width,height:r.height}); }",
      returnByValue: true,
    })) as { result: { value: string } };

    const rect = JSON.parse(result.result.value) as BoundingRect;
    return rect;
  } catch (err) {
    console.error(`[CDP] Error getting bounding rect: ${err}`);
    return null;
  }
}

/**
 * Click an element using CDP Input.dispatchMouseEvent
 */
async function clickElement(cdp: CDPSession, x: number, y: number): Promise<void> {
  // Move mouse to position
  await cdp.send("Input.dispatchMouseEvent", {
    type: "mouseMoved",
    x,
    y,
  });

  // Small delay to show hover state
  await new Promise((r) => setTimeout(r, 50));

  // Mouse down
  await cdp.send("Input.dispatchMouseEvent", {
    type: "mousePressed",
    x,
    y,
    button: "left",
    clickCount: 1,
  });

  // Small delay for click effect
  await new Promise((r) => setTimeout(r, 50));

  // Mouse up
  await cdp.send("Input.dispatchMouseEvent", {
    type: "mouseReleased",
    x,
    y,
    button: "left",
    clickCount: 1,
  });
}

/**
 * Start ffmpeg recording
 */
function startRecording(outputDir: string): Promise<{ pid: number; kill: () => Promise<void> }> {
  return new Promise((resolve, reject) => {
    const outputPath = path.join(outputDir, "raw.mp4");
    const display = process.env.DISPLAY || ":1";

    const ffmpeg = spawn("ffmpeg", [
      "-y",
      "-f", "x11grab",
      "-draw_mouse", "0",
      "-framerate", String(FRAMERATE),
      "-video_size", `${VIDEO_WIDTH}x${VIDEO_HEIGHT}`,
      "-i", `${display}+0,0`,
      "-c:v", "libx264",
      "-preset", "ultrafast",
      "-crf", "26",
      "-pix_fmt", "yuv420p",
      outputPath,
    ], {
      stdio: ["ignore", "pipe", "pipe"],
    });

    let started = false;
    let stderr = "";

    ffmpeg.stderr.on("data", (data) => {
      stderr += data.toString();
      // ffmpeg outputs encoding info to stderr, look for "frame=" to confirm it's recording
      if (!started && (stderr.includes("frame=") || stderr.includes("Output #0"))) {
        started = true;
        console.log("[FFMPEG] Recording started");
        resolve({
          pid: ffmpeg.pid ?? 0,
          kill: () => new Promise<void>((res) => {
            ffmpeg.on("close", () => res());
            ffmpeg.kill("SIGINT");
          }),
        });
      }
    });

    ffmpeg.on("error", (err) => {
      if (!started) reject(err);
    });

    // Give ffmpeg time to start
    setTimeout(() => {
      if (!started) {
        started = true;
        console.log("[FFMPEG] Recording started (assumed)");
        resolve({
          pid: ffmpeg.pid ?? 0,
          kill: () => new Promise<void>((res) => {
            ffmpeg.on("close", () => res());
            ffmpeg.kill("SIGINT");
          }),
        });
      }
    }, 1000);
  });
}

/**
 * Run cursor overlay post-processing (same as in index.ts)
 */
async function runCursorOverlayPostProcessing(outputDir: string): Promise<void> {
  const pythonScript = `
import subprocess, os, sys, json

outdir = "${outputDir}"
events_log_path = f"{outdir}/events.log"

# Parse events.log for click events
# Click events have screen coordinates (with browser chrome offset already applied)
clicks = []  # list of (timestamp_ms, x, y)
recording_start_ms = None

print(f"Reading events from {events_log_path}", file=sys.stderr)

# Parse events.log
line_count = 0
try:
    with open(events_log_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            line_count += 1
            event = json.loads(line)
            ts = event.get("timestamp", 0)
            event_type = event.get("event", "")

            # Detect recording start
            if event_type == "recording_start":
                if recording_start_ms is None:
                    recording_start_ms = ts
                    print(f"Found recording start at {ts}", file=sys.stderr)

            # Click event - has screen coordinates directly
            elif event_type == "click":
                x = event.get("x", 0)
                y = event.get("y", 0)
                clicks.append((ts, x, y))
                print(f"click at ({x}, {y}) ts={ts}", file=sys.stderr)

    print(f"Processed {line_count} events", file=sys.stderr)
except FileNotFoundError:
    print(f"ERROR: events.log not found: {events_log_path}", file=sys.stderr)
except Exception as e:
    print(f"ERROR reading events.log: {e}", file=sys.stderr)

print(f"Found {len(clicks)} clicks", file=sys.stderr)

# Convert timestamps to relative time from first click (cursor starts at center)
if clicks:
    first_ts = clicks[0][0]
    # All clicks relative to first click, starting at t=0.5s (gives time for animation from center)
    clicks = [(0.5 + (ts - first_ts) / 1000.0, x, y) for ts, x, y in clicks]
    print(f"Adjusted clicks: {clicks}", file=sys.stderr)

if clicks:
    filters = []

    # Cursor starts at screen center and animates to first click position
    cx, cy = 960, 540  # screen center
    first_t, first_x, first_y = clicks[0]
    anim_dur = max(first_t, 0.1)  # animation ends at first click (at least 0.1s)

    print(f"Animation: center ({cx},{cy}) -> ({first_x},{first_y}) over {anim_dur:.2f}s", file=sys.stderr)

    # Cursor offsets: yellow outer circle and black center dot
    # Black dot should be centered within yellow circle
    yo_x, yo_y = -14, -20  # yellow offset
    bo_x, bo_y = -6, -12   # black offset (up and left relative to yellow)

    yellow_x_expr = f"({cx+yo_x}+({first_x+yo_x}-{cx+yo_x})*min(t/{anim_dur},1))"
    yellow_y_expr = f"({cy+yo_y}+({first_y+yo_y}-{cy+yo_y})*min(t/{anim_dur},1))"
    black_x_expr = f"({cx+bo_x}+({first_x+bo_x}-{cx+bo_x})*min(t/{anim_dur},1))"
    black_y_expr = f"({cy+bo_y}+({first_y+bo_y}-{cy+bo_y})*min(t/{anim_dur},1))"

    # Animation phase - cursor moves from center to first click
    filters.append(f"drawtext=text='●':x='{yellow_x_expr}':y='{yellow_y_expr}':fontsize=36:fontcolor=yellow@0.5:enable='between(t,0,{anim_dur:.2f})'")
    filters.append(f"drawtext=text='●':x='{black_x_expr}':y='{black_y_expr}':fontsize=12:fontcolor=black:enable='between(t,0,{anim_dur:.2f})'")

    # Static cursor at each click position
    for i, (t, x, y) in enumerate(clicks):
        end_t = clicks[i+1][0] if i+1 < len(clicks) else 9999
        if end_t <= t:
            continue
        e = f"enable='between(t,{t:.2f},{end_t:.2f})'"
        filters.append(f"drawtext=text='●':x={x+yo_x}:y={y+yo_y}:fontsize=36:fontcolor=yellow@0.5:{e}")
        filters.append(f"drawtext=text='●':x={x+bo_x}:y={y+bo_y}:fontsize=12:fontcolor=black:{e}")

    # STEP 1: Draw cursor overlay at original timing (no speed change yet)
    filter_str = ",".join(filters)
    print(f"Drawing cursor overlay with {len(filters)} filter elements", file=sys.stderr)
    result = subprocess.run(f'ffmpeg -y -i "{outdir}/raw.mp4" -vf "{filter_str}" "{outdir}/with_cursor.mp4"', shell=True)
    if result.returncode != 0:
        print(f"Cursor overlay failed with code {result.returncode}", file=sys.stderr)
        os.rename(f"{outdir}/raw.mp4", f"{outdir}/workflow.mp4")
    else:
        os.remove(f"{outdir}/raw.mp4")

        # Get video duration
        probe = subprocess.run(
            ["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", f"{outdir}/with_cursor.mp4"],
            capture_output=True, text=True
        )
        video_duration = float(probe.stdout.strip()) if probe.stdout.strip() else 60.0
        print(f"Video duration: {video_duration:.1f}s", file=sys.stderr)

        # STEP 2: Apply variable speed - 1x during actions, 2x between actions
        # NO TRIMMING - entire video is kept, just at different speeds
        FAST_SPEED = 2
        ACTION_BEFORE = 0.5  # seconds before click at normal speed
        ACTION_AFTER = 0.5   # seconds after click at normal speed

        # Build segments covering the ENTIRE video
        video_segments = []  # (start, end, speed)
        prev_end = 0.0

        for t, x, y in clicks:
            action_start = max(0, t - ACTION_BEFORE)
            action_end = min(video_duration, t + ACTION_AFTER)

            # Fast segment before this action (if there's a gap)
            if action_start > prev_end:
                video_segments.append((prev_end, action_start, FAST_SPEED))

            # Normal speed during action (merge if overlapping with previous)
            if video_segments and video_segments[-1][2] == 1 and action_start <= video_segments[-1][1]:
                # Merge with previous normal-speed segment
                video_segments[-1] = (video_segments[-1][0], action_end, 1)
            else:
                video_segments.append((action_start, action_end, 1))

            prev_end = action_end

        # Add final fast segment after last click (if any video remains)
        if prev_end < video_duration:
            video_segments.append((prev_end, video_duration, FAST_SPEED))

        print(f"Video segments: {video_segments}", file=sys.stderr)

        # Build ffmpeg filter for variable speed
        filter_parts = []
        concat_inputs = []
        for i, (start, end, speed) in enumerate(video_segments):
            pts = 1.0 / speed  # 2x speed = 0.5, 1x = 1.0
            filter_parts.append(f"[0:v]trim=start={start}:end={end},setpts={pts}*(PTS-STARTPTS)[v{i}]")
            concat_inputs.append(f"[v{i}]")

        concat_filter = f"{''.join(concat_inputs)}concat=n={len(video_segments)}:v=1:a=0[out]"
        full_filter = ";".join(filter_parts) + ";" + concat_filter

        print(f"Applying variable speed: 1x during actions, {FAST_SPEED}x between", file=sys.stderr)
        speed_result = subprocess.run([
            "ffmpeg", "-y", "-i", f"{outdir}/with_cursor.mp4",
            "-filter_complex", full_filter,
            "-map", "[out]",
            f"{outdir}/workflow.mp4"
        ])

        if speed_result.returncode != 0:
            print(f"Variable speed failed, using cursor video as-is", file=sys.stderr)
            os.rename(f"{outdir}/with_cursor.mp4", f"{outdir}/workflow.mp4")
        else:
            os.remove(f"{outdir}/with_cursor.mp4")

        # Log final duration
        total_dur = sum((end - start) / speed for start, end, speed in video_segments)
        print(f"Final video: {len(video_segments)} segments, ~{total_dur:.1f}s (from {video_duration:.1f}s original)", file=sys.stderr)

else:
    # No clicks - still draw cursor at center, then speed up
    print("No clicks found, drawing cursor at center and speeding up 2x", file=sys.stderr)
    cx, cy = 960, 540  # screen center
    yo_x, yo_y = -14, -20  # yellow offset
    bo_x, bo_y = -6, -12   # black offset
    cursor_filter = f"drawtext=text='●':x={cx+yo_x}:y={cy+yo_y}:fontsize=36:fontcolor=yellow@0.5,drawtext=text='●':x={cx+bo_x}:y={cy+bo_y}:fontsize=12:fontcolor=black"
    # Draw cursor then speed up
    result = subprocess.run(f'ffmpeg -y -i "{outdir}/raw.mp4" -vf "{cursor_filter},setpts=0.5*PTS" "{outdir}/workflow.mp4"', shell=True)
    if result.returncode == 0:
        os.remove(f"{outdir}/raw.mp4")
    else:
        os.rename(f"{outdir}/raw.mp4", f"{outdir}/workflow.mp4")
`;

  console.log("[POST] Running cursor overlay post-processing...");

  try {
    const { stderr } = await execAsync(`python3 << 'PYSCRIPT'
${pythonScript}
PYSCRIPT`);
    if (stderr) {
      console.log(stderr);
    }
  } catch (err) {
    console.error("[POST] Post-processing error:", err);
    throw err;
  }
}

/**
 * Validate coordinates with tolerance
 */
function coordsMatch(actual: number, expected: number, tolerance = 2): boolean {
  return Math.abs(actual - expected) <= tolerance;
}

/**
 * Main test runner
 */
async function main() {
  console.log("╔" + "═".repeat(62) + "╗");
  console.log("║" + " CDP Bounding Box + Video Recording + Cursor Overlay Test ".padStart(52).padEnd(62) + "║");
  console.log("╚" + "═".repeat(62) + "╝");
  console.log(`\nCDP Port: ${CDP_PORT}`);
  console.log(`Display: ${process.env.DISPLAY || ":1"}`);
  console.log(`Output: ${TEST_OUTPUT_DIR}\n`);

  await fs.mkdir(TEST_OUTPUT_DIR, { recursive: true });

  // Clear events.log
  const eventsLogPath = path.join(TEST_OUTPUT_DIR, "events.log");
  await fs.writeFile(eventsLogPath, "");

  let httpServer: http.Server | null = null;
  let cdp: CDPSession | null = null;
  let recording: { pid: number; kill: () => Promise<void> } | null = null;
  const results: Array<{ element: string; status: string; details: string }> = [];

  try {
    // Step 1: Start test server
    console.log("[1/7] Starting test HTTP server...");
    httpServer = await startTestServer();

    // Step 2: Connect to CDP
    console.log("[2/7] Connecting to Chrome CDP...");
    const target = await getCDPTarget();
    if (!target?.webSocketDebuggerUrl) {
      console.error("\n❌ ERROR: Cannot connect to Chrome.");
      console.error("   Start Chrome with: chromium --remote-debugging-port=9222\n");
      process.exit(1);
    }
    console.log(`   Target: ${target.title || target.url}`);

    cdp = await connectCDP(target.webSocketDebuggerUrl);
    console.log("   Connected to CDP");

    // Enable required domains
    await cdp.send("DOM.enable");
    await cdp.send("Runtime.enable");
    await cdp.send("Page.enable");
    await cdp.send("Input.enable");

    // Step 3: Navigate to test page
    console.log("[3/7] Navigating to test page...");
    await cdp.send("Page.navigate", { url: `http://localhost:${HTTP_PORT}` });
    await new Promise((r) => setTimeout(r, 1000)); // Wait for page load

    // Step 4: Get backendNodeIds for all test elements
    console.log("[4/7] Getting element backendNodeIds...");
    const elementsWithIds: Array<{ id: string; backendNodeId: number; expected: TestElement }> = [];

    for (const el of TEST_ELEMENTS) {
      const backendNodeId = await getBackendNodeId(cdp, `#${el.id}`);
      if (backendNodeId) {
        elementsWithIds.push({ id: el.id, backendNodeId, expected: el });
        console.log(`   ${el.id}: backendNodeId=${backendNodeId}`);
      } else {
        console.error(`   ${el.id}: FAILED to get backendNodeId`);
        results.push({ element: el.id, status: "FAIL", details: "Could not get backendNodeId" });
      }
    }

    // Step 5: Start video recording
    console.log("\n[5/7] Starting video recording...");
    recording = await startRecording(TEST_OUTPUT_DIR);
    console.log(`   Recording PID: ${recording.pid}`);

    // Log recording start event
    const recordingStartTime = Date.now();
    await fs.appendFile(eventsLogPath, JSON.stringify({
      timestamp: recordingStartTime,
      event: "recording_start",
    }) + "\n");

    // Small delay to ensure recording is capturing
    await new Promise((r) => setTimeout(r, 500));

    // Step 6: Click elements and track bounding boxes
    console.log("\n[6/7] Clicking elements and tracking bounding boxes...");
    console.log("─".repeat(60));

    // Only click first 5 elements for demo
    const elementsToClick = elementsWithIds.slice(0, 5);

    for (const el of elementsToClick) {
      // Create uid in the format used by chrome-devtools-mcp
      const uid = `1_${el.backendNodeId}`;

      // Get bounding rect BEFORE clicking
      const rect = await getBoundingRectFromCDP(cdp, uid);

      if (!rect) {
        results.push({ element: el.id, status: "FAIL", details: "getBoundingRectFromCDP returned null" });
        console.log(`   ${el.id}: ❌ FAIL - Could not get bounding rect`);
        continue;
      }

      // Validate coordinates
      const xMatch = coordsMatch(rect.x, el.expected.expectedX);
      const yMatch = coordsMatch(rect.y, el.expected.expectedY);
      const wMatch = coordsMatch(rect.width, el.expected.expectedWidth);
      const hMatch = coordsMatch(rect.height, el.expected.expectedHeight);
      const allMatch = xMatch && yMatch && wMatch && hMatch;

      // Calculate center point for click
      const centerX = rect.x + rect.width / 2;
      const centerY = rect.y + rect.height / 2;

      // Log click event with screen coordinates (add browser chrome offset)
      const clickTimestamp = Date.now();
      const screenX = Math.round(centerX);
      const screenY = Math.round(centerY + BROWSER_CHROME_OFFSET);

      await fs.appendFile(eventsLogPath, JSON.stringify({
        timestamp: clickTimestamp,
        event: "click",
        x: screenX,
        y: screenY,
        uid,
      }) + "\n");

      // Actually click the element
      await clickElement(cdp, centerX, centerY);

      if (allMatch) {
        results.push({
          element: el.id,
          status: "PASS",
          details: `clicked at (${screenX},${screenY})`,
        });
        console.log(`   ${el.id}: ✓ PASS - clicked at screen (${screenX},${screenY})`);
      } else {
        const mismatches: string[] = [];
        if (!xMatch) mismatches.push(`x: got ${rect.x}, expected ${el.expected.expectedX}`);
        if (!yMatch) mismatches.push(`y: got ${rect.y}, expected ${el.expected.expectedY}`);
        if (!wMatch) mismatches.push(`width: got ${rect.width}, expected ${el.expected.expectedWidth}`);
        if (!hMatch) mismatches.push(`height: got ${rect.height}, expected ${el.expected.expectedHeight}`);

        results.push({
          element: el.id,
          status: "WARN",
          details: `clicked but rect mismatch: ${mismatches.join(", ")}`,
        });
        console.log(`   ${el.id}: ⚠ WARN - clicked but rect mismatch`);
      }

      // Delay between clicks for visual effect
      await new Promise((r) => setTimeout(r, 800));
    }

    // Step 7: Stop recording and run post-processing
    console.log("\n[7/7] Stopping recording and applying cursor overlay...");

    if (recording) {
      await recording.kill();
      console.log("   Recording stopped");
    }

    // Run cursor overlay post-processing
    await runCursorOverlayPostProcessing(TEST_OUTPUT_DIR);

    // Print summary
    console.log("\n" + "═".repeat(60));
    console.log("TEST RESULTS:");
    console.log("─".repeat(60));

    const passed = results.filter((r) => r.status === "PASS").length;
    const warned = results.filter((r) => r.status === "WARN").length;
    const failed = results.filter((r) => r.status === "FAIL").length;

    for (const r of results) {
      const icon = r.status === "PASS" ? "✓" : r.status === "WARN" ? "⚠" : "✗";
      console.log(`  ${icon} ${r.element.padEnd(8)} ${r.status.padEnd(6)} ${r.details}`);
    }

    console.log("─".repeat(60));
    console.log(`  Total: ${passed} passed, ${warned} warnings, ${failed} failed out of ${elementsToClick.length} elements`);
    console.log("═".repeat(60));

    // Validate output files
    console.log("\nOutput files:");
    const files = ["events.log", "workflow.mp4"];
    for (const file of files) {
      const filePath = path.join(TEST_OUTPUT_DIR, file);
      try {
        const stats = await fs.stat(filePath);
        console.log(`  ✓ ${file}: ${(stats.size / 1024).toFixed(1)} KB`);
      } catch {
        console.log(`  ✗ ${file}: MISSING`);
      }
    }

    // Show video duration
    try {
      const { stdout } = await execAsync(
        `ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "${path.join(TEST_OUTPUT_DIR, "workflow.mp4")}"`
      );
      console.log(`  Duration: ${parseFloat(stdout.trim()).toFixed(2)}s`);
    } catch {
      // ffprobe not available
    }

    console.log(`\nOutput directory: ${TEST_OUTPUT_DIR}`);

    if (failed === 0) {
      console.log("\n✅ TEST COMPLETED SUCCESSFULLY!\n");
      console.log("The cursor overlay should:");
      console.log("  1. Start at screen center (960, 540)");
      console.log("  2. Animate to first click position");
      console.log("  3. Show yellow circle with black center dot at each click");
      console.log("  4. Speed up (2x) between clicks, normal (1x) during clicks\n");
    } else {
      console.log(`\n⚠ ${failed} TEST(S) FAILED!\n`);
      process.exit(1);
    }

  } catch (err) {
    console.error(`\n❌ FATAL ERROR: ${err}\n`);
    process.exit(1);
  } finally {
    if (recording) {
      try {
        await recording.kill();
      } catch {
        // ignore
      }
    }
    if (cdp) cdp.close();
    if (httpServer) httpServer.close();
  }
}

main();
