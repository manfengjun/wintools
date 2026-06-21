import path from "node:path";

const DEMO_SHORTCUT_NAME = "Wintools演示.lnk";

export function assertSafeDemoShortcut(shortcutPath, desktopPath) {
  const actual = path.win32.resolve(shortcutPath);
  const expected = path.win32.resolve(desktopPath, DEMO_SHORTCUT_NAME);

  if (actual.toLowerCase() !== expected.toLowerCase()) {
    throw new Error(`Unsafe demo shortcut path: ${shortcutPath}`);
  }

  return shortcutPath;
}
