import test from "node:test";
import assert from "node:assert/strict";
import { runDesktopLockScenario } from "../src/desktop-lock-scenario.mjs";

const expectedSteps = [
  "prepareShortcut",
  "launchWintools",
  "enableLock",
  "deleteShortcut",
  "waitForRestore",
  "disableLock",
  "cleanup",
];

function recordingAdapter({ failAt } = {}) {
  const calls = [];
  const adapter = Object.fromEntries(
    expectedSteps.map((name) => [
      name,
      async () => {
        calls.push(name);
        if (name === failAt) throw new Error(`failed at ${name}`);
      },
    ]),
  );
  return { adapter, calls };
}

test("runs the desktop-lock demonstration in deterministic order", async () => {
  const { adapter, calls } = recordingAdapter();
  await runDesktopLockScenario(adapter);
  assert.deepEqual(calls, expectedSteps);
});

test("always cleans up after an intermediate failure", async () => {
  const { adapter, calls } = recordingAdapter({ failAt: "waitForRestore" });
  await assert.rejects(
    () => runDesktopLockScenario(adapter),
    /failed at waitForRestore/,
  );
  assert.deepEqual(calls, [
    "prepareShortcut",
    "launchWintools",
    "enableLock",
    "deleteShortcut",
    "waitForRestore",
    "cleanup",
  ]);
});
