import express, { Request, Response } from "express";
import http from "http";

const app = express();
app.use(express.json());

const COLLECTOR_URL = process.env.COLLECTOR_URL || "http://localhost:8081";
const ANALYZER_URL = process.env.ANALYZER_URL || "http://localhost:8082";
const GATEWAY_PORT = parseInt(process.env.GATEWAY_PORT || "8083", 10);

function log(msg: string): void {
  const ts = new Date().toISOString();
  console.log(`[gateway] ${ts} ${msg}`);
}

function proxyRequest(
  targetUrl: string,
  method: string,
  body?: string
): Promise<{ status: number; data: unknown }> {
  return new Promise((resolve, reject) => {
    const url = new URL(targetUrl);
    const options: http.RequestOptions = {
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      method,
      headers: { "Content-Type": "application/json" },
      timeout: 5000,
    };

    const req = http.request(options, (res) => {
      let data = "";
      res.on("data", (chunk) => (data += chunk));
      res.on("end", () => {
        try {
          resolve({ status: res.statusCode || 200, data: JSON.parse(data) });
        } catch {
          resolve({ status: res.statusCode || 200, data });
        }
      });
    });

    req.on("error", (err) => reject(err));
    req.on("timeout", () => {
      req.destroy();
      reject(new Error("request timeout"));
    });

    if (body) req.write(body);
    req.end();
  });
}

app.get("/health", (_req: Request, res: Response) => {
  res.json({ status: "ok", service: "gateway" });
});

app.get("/api/services", (_req: Request, res: Response) => {
  res.json({
    services: [
      { name: "collector", url: COLLECTOR_URL, description: "Metrics ingestion service" },
      { name: "analyzer", url: ANALYZER_URL, description: "Metrics analytics service" },
      { name: "gateway", url: `http://localhost:${GATEWAY_PORT}`, description: "API gateway" },
    ],
  });
});

app.get("/api/status", async (_req: Request, res: Response) => {
  const services: Record<string, string> = {};

  const checks = [
    { name: "collector", url: `${COLLECTOR_URL}/health` },
    { name: "analyzer", url: `${ANALYZER_URL}/health` },
  ];

  for (const svc of checks) {
    try {
      const result = await proxyRequest(svc.url, "GET");
      services[svc.name] = result.status === 200 ? "healthy" : "unhealthy";
    } catch {
      services[svc.name] = "unreachable";
    }
  }
  services["gateway"] = "healthy";

  log(`Status check: ${JSON.stringify(services)}`);
  res.json({ services });
});

app.post("/api/metrics", async (req: Request, res: Response) => {
  try {
    const result = await proxyRequest(
      `${COLLECTOR_URL}/ingest`,
      "POST",
      JSON.stringify(req.body)
    );
    res.status(result.status).json(result.data);
  } catch (err) {
    log(`Error proxying to collector: ${err}`);
    res.status(502).json({ error: "collector service unavailable" });
  }
});

app.get("/api/metrics", async (_req: Request, res: Response) => {
  try {
    const result = await proxyRequest(`${COLLECTOR_URL}/metrics`, "GET");
    res.status(result.status).json(result.data);
  } catch (err) {
    log(`Error fetching metrics: ${err}`);
    res.status(502).json({ error: "collector service unavailable" });
  }
});

app.get("/api/analyze", async (_req: Request, res: Response) => {
  try {
    const result = await proxyRequest(`${ANALYZER_URL}/analyze`, "GET");
    res.status(result.status).json(result.data);
  } catch (err) {
    log(`Error fetching analysis: ${err}`);
    res.status(502).json({ error: "analyzer service unavailable" });
  }
});

export function createApp() {
  return app;
}

if (require.main === module) {
  app.listen(GATEWAY_PORT, () => {
    log(`Starting gateway service on port ${GATEWAY_PORT}`);
  });
}
