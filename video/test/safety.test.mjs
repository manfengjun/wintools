import test from "node:test";
import assert from "node:assert/strict";
import { assertSafeDemoShortcut } from "../src/safety.mjs";

const desktop = String.raw`C:\Users\Teacher\Desktop`;

test("accepts only the designated demo shortcut on the desktop", () => {
  const shortcut = String.raw`C:\Users\Teacher\Desktop\Wintools演示.lnk`;
  assert.equal(assertSafeDemoShortcut(shortcut, desktop), shortcut);
});

test("compares Windows paths case-insensitively", () => {
  const shortcut = String.raw`c:\users\teacher\desktop\Wintools演示.lnk`;
  assert.equal(
    assertSafeDemoShortcut(shortcut, desktop).toLowerCase(),
    shortcut.toLowerCase(),
  );
});

for (const unsafePath of [
  String.raw`C:\Users\Teacher\Desktop\课程资料.lnk`,
  String.raw`C:\Users\Teacher\Desktop\Wintools演示.txt`,
  String.raw`C:\Users\Teacher\Desktop\folder\..\课程资料.lnk`,
  String.raw`C:\Users\Teacher\Documents\Wintools演示.lnk`,
]) {
  test(`rejects unsafe path: ${unsafePath}`, () => {
    assert.throws(
      () => assertSafeDemoShortcut(unsafePath, desktop),
      /unsafe demo shortcut path/i,
    );
  });
}
