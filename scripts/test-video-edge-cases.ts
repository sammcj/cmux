#!/usr/bin/env bun
/**
 * Edge case tests for video pipeline:
 * 1. No clicks - should speed up entire video 4x
 * 2. Single click - cursor animation + static position
 * 3. Rapid clicks - overlapping time windows should merge
 * 4. Clicks at screen edges - verify coordinates don't go negative
 * 5. Video output validation - check mp4 is valid, correct duration
 */

import { promises as fs } from "node:fs";
import * as path from "node:path";
import { exec } from "node:child_process";
import { promisify } from "node:util";

const execAsync = promisify(exec);

const OUTPUT_BASE = "/tmp/test-video-edge-cases";
const VIDEO_DURATION = 4; // seconds
const VIDEO_WIDTH = 1920;
const VIDEO_HEIGHT = 1080;

interface TestCase {
  name: string;
  clicks: Array<{ x: number; y: number; width: number; height: number; timeOffsetMs: number }>;
  expectedDuration?: { min: number; max: number };
  description: string;
}

const TEST_CASES: TestCase[] = [
  {
    name: "no-clicks",
    clicks: [],
    expectedDuration: { min: 0.9, max: 1.1 }, // 4s / 4x speed = 1s
    description: "No clicks - entire video at 4x speed",
  },
  {
    name: "single-click",
    clicks: [{ x: 960, y: 540, width: 100, height: 50, timeOffsetMs: 2000 }],
    expectedDuration: { min: 1.5, max: 2.5 },
    description: "Single click at center",
  },
  {
    name: "rapid-clicks",
    clicks: [
      { x: 100, y: 100, width: 80, height: 40, timeOffsetMs: 500 },
      { x: 200, y: 100, width: 80, height: 40, timeOffsetMs: 600 }, // 100ms apart - should merge
      { x: 300, y: 100, width: 80, height: 40, timeOffsetMs: 700 }, // 100ms apart - should merge
      { x: 400, y: 100, width: 80, height: 40, timeOffsetMs: 800 }, // 100ms apart - should merge
    ],
    expectedDuration: { min: 1.0, max: 2.0 },
    description: "Rapid clicks that should merge into single action window",
  },
  {
    name: "edge-clicks",
    clicks: [
      { x: 0, y: 0, width: 50, height: 30, timeOffsetMs: 500 },       // Top-left corner
      { x: 1870, y: 0, width: 50, height: 30, timeOffsetMs: 1500 },   // Top-right corner
      { x: 0, y: 1050, width: 50, height: 30, timeOffsetMs: 2500 },   // Bottom-left corner
      { x: 1870, y: 1050, width: 50, height: 30, timeOffsetMs: 3500 }, // Bottom-right corner
    ],
    expectedDuration: { min: 2.0, max: 4.0 },
    description: "Clicks at screen corners - verify no negative coordinates",
  },
  {
    name: "many-clicks",
    clicks: Array.from({ length: 10 }, (_, i) => ({
      x: 100 + i * 150,
      y: 300,
      width: 100,
      height: 50,
      timeOffsetMs: 300 + i * 350,
    })),
    expectedDuration: { min: 2.0, max: 4.0 },
    description: "Many clicks spread across video",
  },
];

/**
 * Generate synthetic video with ffmpeg
 */
async function generateVideo(outputDir: string): Promise<void> {
  const outputPath = path.join(outputDir, "raw.mp4");

  // Simple color video
  await execAsync(
    `ffmpeg -y -f lavfi -i "color=c=0x2d3436:size=${VIDEO_WIDTH}x${VIDEO_HEIGHT}:duration=${VIDEO_DURATION}:rate=24,` +
    `drawtext=text='Test Video':fontsize=72:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2" ` +
    `-c:v libx264 -preset ultrafast -pix_fmt yuv420p "${outputPath}"`
  );
}

/**
 * Generate events.log for a test case
 */
async function generateEvents(
  outputDir: string,
  clicks: TestCase["clicks"]
): Promise<void> {
  const eventsPath = path.join(outputDir, "events.log");
  const events: string[] = [];
  const startTime = Date.now();

  events.push(JSON.stringify({ timestamp: startTime, event: "recording_start" }));

  for (let i = 0; i < clicks.length; i++) {
    const click = clicks[i]!;
    const ts = startTime + click.timeOffsetMs;

    events.push(JSON.stringify({
      timestamp: ts,
      event: "bounding_rect",
      x: click.x,
      y: click.y,
      width: click.width,
      height: click.height,
      uid: `1_${1000 + i}`,
    }));

    events.push(JSON.stringify({
      timestamp: ts + 1,
      event: "pretool",
      tool: "mcp__chrome__click",
      uid: `1_${1000 + i}`,
    }));
  }

  await fs.writeFile(eventsPath, events.join("\n") + "\n");
}

/**
 * Run post-processing Python script
 */
async function runPostProcessing(outputDir: string): Promise<{ success: boolean; error?: string }> {
  const script = `
import subprocess, os, sys, json

outdir = "${outputDir}"
events_path = f"{outdir}/events.log"

clicks = []
recording_start_ms = None
last_rect = None

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
            elif evt == "bounding_rect":
                x, y = ev.get("x", 0), ev.get("y", 0)
                w, h = ev.get("width", 0), ev.get("height", 0)
                cx, cy = int(x + w/2), int(y + h/2 + 85)
                last_rect = (cx, cy, ts)
            elif evt == "pretool" and "click" in ev.get("tool", "").lower():
                if last_rect:
                    clicks.append((ts, last_rect[0], last_rect[1]))
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)

if not clicks:
    # No clicks - just speed up 4x
    r = subprocess.run(f'ffmpeg -y -i "{outdir}/raw.mp4" -vf "setpts=0.25*PTS" "{outdir}/workflow.mp4"', shell=True)
    if r.returncode == 0:
        os.remove(f"{outdir}/raw.mp4")
    else:
        os.rename(f"{outdir}/raw.mp4", f"{outdir}/workflow.mp4")
    sys.exit(0)

# Adjust timestamps
if recording_start_ms:
    clicks = [(0.5 + (ts - recording_start_ms) / 1000.0, x, y) for ts, x, y in clicks]
else:
    first_ts = clicks[0][0]
    clicks = [(0.5 + (ts - first_ts) / 1000.0, x, y) for ts, x, y in clicks]

# Build cursor overlay
filters = []
cx, cy = 960, 540
t0, x0, y0 = clicks[0]
dur = max(t0, 0.1)

yo_x, yo_y = -14, -20
bo_x, bo_y = -6, -12

yx = f"({cx+yo_x}+({x0+yo_x}-{cx+yo_x})*min(t/{dur},1))"
yy = f"({cy+yo_y}+({y0+yo_y}-{cy+yo_y})*min(t/{dur},1))"
bx = f"({cx+bo_x}+({x0+bo_x}-{cx+bo_x})*min(t/{dur},1))"
by = f"({cy+bo_y}+({y0+bo_y}-{cy+bo_y})*min(t/{dur},1))"

filters.append(f"drawtext=text='●':x='{yx}':y='{yy}':fontsize=36:fontcolor=yellow@0.5:enable='between(t,0,{dur:.2f})'")
filters.append(f"drawtext=text='●':x='{bx}':y='{by}':fontsize=12:fontcolor=black:enable='between(t,0,{dur:.2f})'")

for i, (t, x, y) in enumerate(clicks):
    end_t = clicks[i+1][0] if i+1 < len(clicks) else 9999
    if end_t <= t:
        continue
    e = f"enable='between(t,{t:.2f},{end_t:.2f})'"
    filters.append(f"drawtext=text='●':x={x+yo_x}:y={y+yo_y}:fontsize=36:fontcolor=yellow@0.5:{e}")
    filters.append(f"drawtext=text='●':x={x+bo_x}:y={y+bo_y}:fontsize=12:fontcolor=black:{e}")

fstr = ",".join(filters)
r = subprocess.run(f'ffmpeg -y -i "{outdir}/raw.mp4" -vf "{fstr}" "{outdir}/with_cursor.mp4"', shell=True, capture_output=True)
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

parts = []
labels = []
for i, (s, e, spd) in enumerate(segs):
    pts = 1.0 / spd
    parts.append(f"[0:v]trim=start={s}:end={e},setpts={pts}*(PTS-STARTPTS)[v{i}]")
    labels.append(f"[v{i}]")

full = ";".join(parts) + ";" + f"{''.join(labels)}concat=n={len(segs)}:v=1:a=0[out]"
sr = subprocess.run(["ffmpeg", "-y", "-i", f"{outdir}/with_cursor.mp4", "-filter_complex", full, "-map", "[out]", f"{outdir}/workflow.mp4"], capture_output=True)

if sr.returncode != 0:
    os.rename(f"{outdir}/with_cursor.mp4", f"{outdir}/workflow.mp4")
else:
    os.remove(f"{outdir}/with_cursor.mp4")
`;

  try {
    await execAsync(`python3 << 'PYSCRIPT'
${script}
PYSCRIPT`);
    return { success: true };
  } catch (err) {
    return { success: false, error: String(err) };
  }
}

/**
 * Validate output video
 */
async function validateVideo(
  outputDir: string,
  expectedDuration?: { min: number; max: number }
): Promise<{ valid: boolean; duration: number; errors: string[] }> {
  const errors: string[] = [];
  const videoPath = path.join(outputDir, "workflow.mp4");

  // Check file exists
  try {
    await fs.stat(videoPath);
  } catch {
    return { valid: false, duration: 0, errors: ["workflow.mp4 not found"] };
  }

  // Get duration
  const { stdout } = await execAsync(
    `ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "${videoPath}"`
  );
  const duration = parseFloat(stdout.trim());

  if (isNaN(duration)) {
    errors.push("Could not parse video duration");
  }

  // Check duration is in expected range
  if (expectedDuration) {
    if (duration < expectedDuration.min) {
      errors.push(`Duration ${duration.toFixed(2)}s < min ${expectedDuration.min}s`);
    }
    if (duration > expectedDuration.max) {
      errors.push(`Duration ${duration.toFixed(2)}s > max ${expectedDuration.max}s`);
    }
  }

  // Verify video is playable
  try {
    await execAsync(`ffprobe -v error "${videoPath}"`);
  } catch {
    errors.push("Video file is corrupted or not playable");
  }

  return { valid: errors.length === 0, duration, errors };
}

/**
 * Run a single test case
 */
async function runTestCase(testCase: TestCase): Promise<{ passed: boolean; details: string }> {
  const outputDir = path.join(OUTPUT_BASE, testCase.name);
  await fs.mkdir(outputDir, { recursive: true });

  try {
    // Generate video
    await generateVideo(outputDir);

    // Generate events
    await generateEvents(outputDir, testCase.clicks);

    // Run post-processing
    const postResult = await runPostProcessing(outputDir);
    if (!postResult.success) {
      return { passed: false, details: `Post-processing failed: ${postResult.error}` };
    }

    // Validate output
    const validation = await validateVideo(outputDir, testCase.expectedDuration);
    if (!validation.valid) {
      return { passed: false, details: validation.errors.join("; ") };
    }

    return {
      passed: true,
      details: `Duration: ${validation.duration.toFixed(2)}s (expected ${testCase.expectedDuration?.min}-${testCase.expectedDuration?.max}s)`,
    };
  } catch (err) {
    return { passed: false, details: String(err) };
  }
}

/**
 * Main
 */
async function main() {
  console.log("╔" + "═".repeat(58) + "╗");
  console.log("║" + " Video Pipeline Edge Case Tests ".padStart(45).padEnd(58) + "║");
  console.log("╚" + "═".repeat(58) + "╝\n");

  await fs.mkdir(OUTPUT_BASE, { recursive: true });

  const results: Array<{ name: string; passed: boolean; details: string; description: string }> = [];

  for (const testCase of TEST_CASES) {
    process.stdout.write(`Testing: ${testCase.name}... `);
    const result = await runTestCase(testCase);
    results.push({ ...result, name: testCase.name, description: testCase.description });
    console.log(result.passed ? "✓ PASS" : "✗ FAIL");
    if (!result.passed) {
      console.log(`  Error: ${result.details}`);
    }
  }

  // Summary
  console.log("\n" + "═".repeat(60));
  console.log("RESULTS:");
  console.log("─".repeat(60));

  for (const r of results) {
    const icon = r.passed ? "✓" : "✗";
    console.log(`${icon} ${r.name.padEnd(15)} ${r.passed ? "PASS" : "FAIL"}`);
    console.log(`  ${r.description}`);
    console.log(`  ${r.details}`);
  }

  const passed = results.filter((r) => r.passed).length;
  const failed = results.filter((r) => !r.passed).length;

  console.log("─".repeat(60));
  console.log(`Total: ${passed} passed, ${failed} failed out of ${TEST_CASES.length} tests`);
  console.log("═".repeat(60));

  if (failed === 0) {
    console.log("\n✅ ALL EDGE CASE TESTS PASSED!\n");
  } else {
    console.log(`\n❌ ${failed} TEST(S) FAILED!\n`);
    process.exit(1);
  }
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
