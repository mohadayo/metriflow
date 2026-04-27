import request from "supertest";
import { createApp } from "./index";

const app = createApp();

describe("Gateway Health", () => {
  it("should return health status", async () => {
    const res = await request(app).get("/health");
    expect(res.status).toBe(200);
    expect(res.body.status).toBe("ok");
    expect(res.body.service).toBe("gateway");
  });
});

describe("GET /api/services", () => {
  it("should list all services", async () => {
    const res = await request(app).get("/api/services");
    expect(res.status).toBe(200);
    expect(res.body.services).toHaveLength(3);
    const names = res.body.services.map((s: { name: string }) => s.name);
    expect(names).toContain("collector");
    expect(names).toContain("analyzer");
    expect(names).toContain("gateway");
  });
});

describe("POST /api/metrics", () => {
  it("should return 502 when collector is down", async () => {
    const res = await request(app)
      .post("/api/metrics")
      .send([{ name: "test", value: 1 }]);
    expect(res.status).toBe(502);
    expect(res.body.error).toBeDefined();
  });
});

describe("GET /api/metrics", () => {
  it("should return 502 when collector is down", async () => {
    const res = await request(app).get("/api/metrics");
    expect(res.status).toBe(502);
  });
});

describe("GET /api/analyze", () => {
  it("should return 502 when analyzer is down", async () => {
    const res = await request(app).get("/api/analyze");
    expect(res.status).toBe(502);
  });
});

describe("GET /api/status", () => {
  it("should report service statuses", async () => {
    const res = await request(app).get("/api/status");
    expect(res.status).toBe(200);
    expect(res.body.services).toBeDefined();
    expect(res.body.services.gateway).toBe("healthy");
    expect(res.body.services.collector).toBe("unreachable");
    expect(res.body.services.analyzer).toBe("unreachable");
  });
});
