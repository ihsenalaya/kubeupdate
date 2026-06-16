import os
from contextlib import asynccontextmanager

import asyncpg
from fastapi import FastAPI, HTTPException

POSTGRES_DSN = os.environ["POSTGRES_DSN"]

pool: asyncpg.Pool | None = None


@asynccontextmanager
async def lifespan(_: FastAPI):
    global pool
    pool = await asyncpg.create_pool(POSTGRES_DSN, min_size=1, max_size=5)
    async with pool.acquire() as conn:
        await conn.execute(
            """
            create table if not exists products (
              id serial primary key,
              sku text not null unique,
              name text not null,
              category text not null,
              price numeric(10, 2) not null
            )
            """
        )
        await conn.execute(
            """
            insert into products (sku, name, category, price)
            values
              ('AKS-OPS-001', 'Private AKS Readiness Review', 'platform', 1200.00),
              ('OBS-OTEL-002', 'OpenTelemetry Signal Pack', 'observability', 450.00),
              ('SEC-KV-003', 'Key Vault Certificate Audit', 'security', 300.00)
            on conflict (sku) do nothing
            """
        )
    yield
    await pool.close()


app = FastAPI(title="Upgrade Lab Catalog", version="0.1.0", lifespan=lifespan)


@app.get("/healthz")
async def healthz():
    return {"status": "ok", "service": "catalog-service"}


@app.get("/readyz")
async def readyz():
    try:
        async with pool.acquire() as conn:
            await conn.fetchval("select 1")
        return {"status": "ready"}
    except Exception as exc:  # pragma: no cover - exercised in cluster
        raise HTTPException(status_code=503, detail=str(exc)) from exc


@app.get("/products")
async def products():
    async with pool.acquire() as conn:
        rows = await conn.fetch("select sku, name, category, price::text from products order by sku")
    return [dict(row) for row in rows]
