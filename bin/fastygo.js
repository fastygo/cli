#!/usr/bin/env node
// Downloads the fastygo release binary for this OS and runs it.
// npx, pnpm dlx, yarn dlx, and bunx all use this file.

import { execFileSync, spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { chmodSync, createWriteStream, existsSync, mkdirSync, readFileSync } from "node:fs";
import { homedir, platform, arch } from "node:os";
import { join } from "node:path";
import { pipeline } from "node:stream/promises";

const version = "0.1.0";
const repo = "fastygo/cli";

const targets = {
  "win32-x64": { file: `fastygo_${version}_windows_amd64.zip`, bin: "fastygo.exe" },
  "linux-x64": { file: `fastygo_${version}_linux_amd64.tar.gz`, bin: "fastygo" },
  "darwin-arm64": { file: `fastygo_${version}_darwin_arm64.tar.gz`, bin: "fastygo" },
};

const key = `${platform()}-${arch()}`;
const target = targets[key];
if (!target) {
  console.error(`fastygo ${version} publishes binaries for windows/amd64, linux/amd64, and darwin/arm64 (this system is ${key}).`);
  console.error(`From Go: go install github.com/fastygo/cli/cmd/fastygo@v${version}`);
  process.exit(1);
}

const cache = join(homedir(), ".cache", "fastygo", version);
const binary = join(cache, target.bin);
if (!existsSync(binary)) {
  mkdirSync(cache, { recursive: true });
  const base = `https://github.com/${repo}/releases/download/v${version}`;
  const archive = join(cache, target.file);
  await download(`${base}/${target.file}`, archive);
  await verify(`${base}/SHA256SUMS`, target.file, archive);
  await unpack(archive, cache, target.bin);
  if (platform() !== "win32") chmodSync(binary, 0o755);
}

const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" });
child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  process.exit(code ?? 1);
});

async function download(url, dest) {
  const response = await fetch(url);
  if (!response.ok) {
    console.error(`Could not download ${url} (${response.status}).`);
    console.error(`From Go: go install github.com/fastygo/cli/cmd/fastygo@v${version}`);
    process.exit(1);
  }
  await pipeline(response.body, createWriteStream(dest));
}

async function verify(url, filename, file) {
  const response = await fetch(url);
  if (!response.ok) {
    console.error(`Could not download checksums (${response.status}). Refusing to run an unverified binary.`);
    process.exit(1);
  }
  const sums = await response.text();
  const line = sums.split(/\r?\n/).find((entry) => entry.endsWith(`  ${filename}`) || entry.endsWith(` ${filename}`));
  if (!line) {
    console.error(`SHA256SUMS has no entry for ${filename}.`);
    process.exit(1);
  }
  const want = line.split(/\s+/)[0];
  const got = createHash("sha256").update(readFileSync(file)).digest("hex");
  if (got !== want) {
    console.error(`checksum mismatch for ${filename}`);
    process.exit(1);
  }
}

async function unpack(archive, dest, bin) {
  if (archive.endsWith(".zip") && platform() === "win32") {
    execFileSync("powershell", ["-NoProfile", "-Command", `Expand-Archive -Force -Path '${archive}' -DestinationPath '${dest}'`], { stdio: "inherit" });
  } else {
    execFileSync("tar", ["-xzf", archive, "-C", dest], { stdio: "inherit" });
  }
  if (!existsSync(join(dest, bin))) {
    console.error(`archive did not contain ${bin}`);
    process.exit(1);
  }
}
