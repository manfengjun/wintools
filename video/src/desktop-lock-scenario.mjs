const SCENE_STEPS = [
  "prepareShortcut",
  "launchWintools",
  "enableLock",
  "deleteShortcut",
  "waitForRestore",
  "disableLock",
];

export async function runDesktopLockScenario(adapter) {
  try {
    for (const step of SCENE_STEPS) {
      await adapter[step]();
    }
  } finally {
    await adapter.cleanup();
  }
}
