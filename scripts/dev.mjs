#!/usr/bin/env node
/**
 * One-command desktop launcher: npm run dev / bun run dev
 *
 * Windows wails3/task need Git's uname/tail and amd64 Go on PATH.
 * A leftover Vite/Wails process on 9245 will otherwise fail the bind.
 */
import { spawn, execFileSync, execSync } from "node:child_process";
import { existsSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const win = process.platform === "win32";
const port = Number(process.env.WAILS_VITE_PORT) || 9245;

function prependPath(dir) {
  if (!dir || !existsSync(dir)) return;
  const parts = (process.env.PATH || "").split(path.delimiter);
  if (parts.some((p) => p.toLowerCase() === dir.toLowerCase())) return;
  process.env.PATH = dir + path.delimiter + process.env.PATH;
}

function setupPath() {
  if (win) {
    prependPath("C:\\Program Files\\Go\\bin");
    prependPath(path.join(os.homedir(), "go", "bin"));
    prependPath("C:\\Program Files\\Git\\usr\\bin");
    prependPath("C:\\Program Files\\Git\\bin");
    prependPath("C:\\Program Files\\nodejs");
  } else {
    prependPath("/usr/local/go/bin");
    prependPath(path.join(os.homedir(), "go", "bin"));
  }
}

function pidsOnPort(p) {
  const ids = new Set();
  try {
    if (win) {
      const out = execSync("netstat -ano", { encoding: "utf8" });
      const re = new RegExp(`[:\\[]${p}\\]?\\s+\\S+\\s+LISTENING\\s+(\\d+)`, "gi");
      let m;
      while ((m = re.exec(out))) {
        const pid = Number(m[1]);
        if (pid > 4) ids.add(pid);
      }
    } else {
      const out = execFileSync("lsof", ["-ti", `tcp:${p}`], { encoding: "utf8" }).trim();
      for (const line of out.split(/\s+/)) {
        const pid = Number(line);
        if (pid > 1) ids.add(pid);
      }
    }
  } catch {
    /* empty */
  }
  return [...ids];
}

function killPid(pid) {
  try {
    if (win) execSync(`taskkill /F /PID ${pid}`, { stdio: "ignore" });
    else process.kill(pid, "SIGTERM");
  } catch {
    /* already gone */
  }
}

function killNamed(name) {
  if (!win) return;
  try {
    execSync(`taskkill /F /IM ${name}`, { stdio: "ignore" });
  } catch {
    /* not running */
  }
}

function freePort(p) {
  killNamed("Yoyo.exe");
  for (const pid of pidsOnPort(p)) {
    if (pid === process.pid) continue;
    killPid(pid);
  }
}

function hasCmd(cmd) {
  try {
    if (win) execSync(`where ${cmd}`, { stdio: "ignore" });
    else execFileSync("which", [cmd], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function run(cmd, args) {
  const child = spawn(cmd, args, {
    cwd: root,
    env: process.env,
    stdio: "inherit",
    shell: win,
  });
  child.on("exit", (code, signal) => {
    if (signal) process.exit(1);
    process.exit(code ?? 1);
  });
}

setupPath();
freePort(port);

if (!hasCmd("wails3")) {
  console.error("wails3 not found. Install once:");
  console.error("  go install github.com/wailsapp/wails/v3/cmd/wails3@latest");
  process.exit(1);
}

if (!hasCmd("go")) {
  console.error("go not found. Install the amd64 toolchain and retry.");
  process.exit(1);
}

process.env.WAILS_VITE_PORT = String(port);
console.log(`Yoyo desktop  →  wails3 task dev  (vite :${port})`);
run("wails3", ["task", "dev"]);
