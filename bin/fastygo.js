#!/usr/bin/env node
// Runs the fastygo binary shipped in this package.
// npx, pnpm dlx, yarn dlx, and bunx all use this file.

import { spawn } from "node:child_process";
import { chmodSync, existsSync } from "node:fs";
import { arch, platform } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const version = "0.1.1";

const packed = {
  "win32-x64": "fastygo-windows-amd64.exe",
  "linux-x64": "fastygo-linux-amd64",
  "darwin-arm64": "fastygo-darwin-arm64",
};

const key = `${platform()}-${arch()}`;
const name = packed[key];
if (!name) {
  console.error(`fastygo ${version} publishes binaries for windows/amd64, linux/amd64, and darwin/arm64 (this system is ${key}).`);
  process.exit(1);
}

const binary = join(dirname(fileURLToPath(import.meta.url)), "..", "binaries", name);
if (!existsSync(binary)) {
  console.error(`missing ${binary}`);
  console.error(`From a checkout of the module: go install github.com/fastygo/cli/cmd/fastygo@v${version}`);
  process.exit(1);
}
if (platform() !== "win32") chmodSync(binary, 0o755);

const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  process.exit(code ?? 1);
});
