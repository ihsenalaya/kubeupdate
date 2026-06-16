import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import Fastify from "fastify";
import Redis from "ioredis";
import { request } from "undici";

const port = Number.parseInt(process.env.PORT ?? "3000", 10);
const catalogUrl = process.env.CATALOG_URL ?? "http://catalog:8000";
const ordersUrl = process.env.ORDERS_URL ?? "http://orders:8080";
const signalsUrl = process.env.SIGNALS_URL ?? "http://signals:8090";
const redisUrl = process.env.REDIS_URL;
const certPath = process.env.CLIENT_CERT_PATH ?? "/etc/lab-certs/client-certificate.pfx";

const app = Fastify({ logger: true });
const redis = redisUrl ? new Redis(redisUrl, { lazyConnect: true, maxRetriesPerRequest: 2 }) : undefined;

async function getJson(url) {
  const response = await request(url, { method: "GET", headersTimeout: 5000, bodyTimeout: 5000 });
  const body = await response.body.text();
  if (response.statusCode >= 400) {
    throw new Error(`GET ${url} failed with ${response.statusCode}: ${body}`);
  }
  return JSON.parse(body);
}

async function cached(key, ttlSeconds, producer) {
  if (!redis) {
    return producer();
  }
  if (redis.status === "wait") {
    await redis.connect();
  }
  const hit = await redis.get(key);
  if (hit) {
    return { source: "redis", data: JSON.parse(hit) };
  }
  const data = await producer();
  await redis.set(key, JSON.stringify(data), "EX", ttlSeconds);
  return { source: "origin", data };
}

app.get("/healthz", async () => ({ status: "ok", service: "edge-api" }));

app.get("/readyz", async (_request, reply) => {
  try {
    if (redis) {
      if (redis.status === "wait") {
        await redis.connect();
      }
      await redis.ping();
    }
    return { status: "ready" };
  } catch (error) {
    reply.code(503);
    return { status: "not-ready", error: error.message };
  }
});

app.get("/", async () => ({
  name: "Upgrade Lab",
  services: ["edge-api", "catalog-service", "orders-service", "signals-service"],
  dataStores: ["Azure PostgreSQL", "Azure SQL Database", "Azure Cosmos DB Mongo API", "Azure Cache for Redis", "Azure Key Vault"]
}));

app.get("/api/products", async () => cached("catalog:products", 30, () => getJson(`${catalogUrl}/products`)));
app.get("/api/orders", async () => getJson(`${ordersUrl}/orders`));
app.get("/api/signals", async () => getJson(`${signalsUrl}/signals`));

app.get("/api/summary", async () => {
  const [products, orders, signals] = await Promise.all([
    cached("catalog:products", 30, () => getJson(`${catalogUrl}/products`)),
    getJson(`${ordersUrl}/orders`),
    getJson(`${signalsUrl}/signals`)
  ]);
  return { products, orders, signals };
});

app.get("/api/certificate", async (_request, reply) => {
  try {
    const certificate = await readFile(certPath);
    return {
      mounted: true,
      bytes: certificate.length,
      sha256: createHash("sha256").update(certificate).digest("hex")
    };
  } catch (error) {
    reply.code(503);
    return { mounted: false, error: error.message };
  }
});

app.listen({ host: "0.0.0.0", port });
