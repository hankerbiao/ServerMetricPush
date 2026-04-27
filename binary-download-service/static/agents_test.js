const fs = require("fs");
const path = require("path");
const vm = require("vm");

class Element {
  constructor(id = "") {
    this.id = id;
    this.innerHTML = "";
    this.listeners = {};
    this.style = {};
    this.value = "";
    this.disabled = false;
    this.dataset = {};
  }

  addEventListener(type, handler) {
    if (!this.listeners[type]) {
      this.listeners[type] = [];
    }
    this.listeners[type].push(handler);
  }

  querySelectorAll() {
    return [];
  }
}

function extractScript(html) {
  const match = html.match(/<script>([\s\S]*)<\/script>\s*<\/body>/);
  if (!match) {
    throw new Error("failed to extract inline script");
  }
  return match[1];
}

async function main() {
  const htmlPath = path.join(__dirname, "agents.html");
  const html = fs.readFileSync(htmlPath, "utf8");
  const script = extractScript(html);

  const elements = new Map();
  const document = {
    getElementById(id) {
      if (!elements.has(id)) {
        elements.set(id, new Element(id));
      }
      return elements.get(id);
    }
  };

  const context = vm.createContext({
    console,
    document,
    window: {
      location: { origin: "http://control-plane.local" },
      crypto: { randomUUID: () => "req-123" }
    },
    fetch: async (url) => {
      if (url === "/api/agents") {
        return {
          ok: true,
          async json() {
            return { agents: [] };
          }
        };
      }
      if (url === "/api/files?program=node-push-exporter") {
        return {
          ok: true,
          async json() {
            return { files: [] };
          }
        };
      }
      if (url.startsWith("/api/config-templates/")) {
        return {
          ok: true,
          async text() {
            return JSON.stringify({ id: 7, version: "cfg-v2", content: "k=v" });
          }
        };
      }
      throw new Error(`unexpected fetch: ${url}`);
    },
    setInterval() {},
    Date,
    URL,
    Math,
    JSON
  });

  vm.runInContext(script, context, { filename: htmlPath });
  await new Promise((resolve) => setImmediate(resolve));

  const baseURL = vm.runInContext(
    `buildAgentUpdateBaseURL({ update_listen_addr: "10.0.0.5:18080", ip: "10.0.0.5" })`,
    context
  );
  if (baseURL !== "http://10.0.0.5:18080") {
    throw new Error(`unexpected base URL: ${baseURL}`);
  }

  const translatedBaseURL = vm.runInContext(
    `buildAgentUpdateBaseURL({ update_listen_addr: "127.0.0.1:18080", ip: "10.0.0.9" })`,
    context
  );
  if (translatedBaseURL !== "http://10.0.0.9:18080") {
    throw new Error(`unexpected translated base URL: ${translatedBaseURL}`);
  }

  const translatedURLBase = vm.runInContext(
    `buildAgentUpdateBaseURL({ update_listen_addr: "http://127.0.0.1:18080", ip: "10.0.0.11" })`,
    context
  );
  if (translatedURLBase !== "http://10.0.0.11:18080") {
    throw new Error(`unexpected translated URL base: ${translatedURLBase}`);
  }

  const binaryPayload = vm.runInContext(
    `JSON.stringify(buildBinaryUpdatePayload({ version: "1.2.4", filename: "node-push-exporter-1.2.4-linux-amd64.tar.gz" }))`,
    context
  );
  const parsedBinary = JSON.parse(binaryPayload);
  if (parsedBinary.request_id !== "req-123") {
    throw new Error(`unexpected request id: ${parsedBinary.request_id}`);
  }
  if (parsedBinary.download_url !== "http://control-plane.local/download/node-push-exporter-1.2.4-linux-amd64.tar.gz") {
    throw new Error(`unexpected download url: ${parsedBinary.download_url}`);
  }
  if (parsedBinary.package_type !== "tar.gz") {
    throw new Error(`unexpected package type: ${parsedBinary.package_type}`);
  }

  const configPayload = vm.runInContext(
    `JSON.stringify(buildConfigUpdatePayload({ id: 7, version: "cfg-v2", content: "k=v" }))`,
    context
  );
  const parsedConfig = JSON.parse(configPayload);
  if (parsedConfig.config_template_id !== "7") {
    throw new Error(`unexpected template id: ${parsedConfig.config_template_id}`);
  }
  if (parsedConfig.config_content !== "k=v") {
    throw new Error(`unexpected config content: ${parsedConfig.config_content}`);
  }

  const filteredVersions = vm.runInContext(
    `JSON.stringify(filterBinaryVersions({ files: [
      { id: 1, filename: "node-push-exporter-1.2.4-linux-amd64.tar.gz", program: "node-push-exporter", version: "1.2.4", os: "linux", arch: "amd64" },
      { id: 2, filename: "node-push-exporter-1.2.4-darwin-arm64.tar.gz", program: "node-push-exporter", version: "1.2.4", os: "darwin", arch: "arm64" },
      { id: 3, filename: "node_exporter-1.10.2-linux-amd64.tar.gz", program: "node_exporter", version: "1.10.2", os: "linux", arch: "amd64" }
    ]}, { os: "linux", arch: "amd64" }))`,
    context
  );
  const parsedVersions = JSON.parse(filteredVersions);
  if (parsedVersions.length !== 1) {
    throw new Error(`unexpected filtered versions length: ${parsedVersions.length}`);
  }
  if (parsedVersions[0].filename !== "node-push-exporter-1.2.4-linux-amd64.tar.gz") {
    throw new Error(`unexpected filtered version filename: ${parsedVersions[0].filename}`);
  }

  console.log("agents frontend tests passed");
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
