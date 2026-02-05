const { mkdirSync, createWriteStream } = require("node:fs");
const { join } = require("node:path");
const https = require("node:https");

const pkg = require("../package.json");
const version = `v${pkg.version}`;
const platform = process.platform;
const arch = process.arch;

const map = {
  "linux-x64": "remdoc-linux-amd64",
  "linux-arm64": "remdoc-linux-arm64",
  "darwin-x64": "remdoc-darwin-amd64",
  "darwin-arm64": "remdoc-darwin-arm64",
  "win32-x64": "remdoc-windows-amd64.exe",
};

const key = `${platform}-${arch}`;
const asset = map[key];

if (!asset) {
  console.error(`Unsupported platform: ${platform} ${arch}`);
  process.exit(1);
}

const url = `https://github.com/Elias-Larsson/remdoc/releases/download/${version}/${asset}`;
const outDir = join(__dirname, "..", "bin");
const outFile = join(outDir, platform === "win32" ? "remdoc.exe" : "remdoc");

mkdirSync(outDir, { recursive: true });

function download(url, attempt = 0) {
  if (attempt > 5) {
    console.error("Too many redirects");
    process.exit(1);
  }

  https
    .get(url, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        download(res.headers.location, attempt + 1);
        return;
      }

      if (res.statusCode !== 200) {
        console.error(`Download failed: ${res.statusCode}`);
        process.exit(1);
      }

      const file = createWriteStream(outFile, { mode: 0o755 });
      res.pipe(file);
      file.on("finish", () => {
        file.close();
        console.log("✓ Binary downloaded successfully");
      });
    })
    .on("error", (err) => {
      console.error(`Download error: ${err.message}`);
      process.exit(1);
    });
}

download(url);
