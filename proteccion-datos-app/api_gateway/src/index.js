import cors from "cors";
import express from "express";
import helmet from "helmet";
import { createProxyMiddleware } from "http-proxy-middleware";
import pg from "pg";

const app = express();
const PORT = process.env.PORT || 4000;
const AI_SERVICE_URL = process.env.AI_SERVICE_URL || "http://localhost:8000";
const DEMO_TENANT_ID = "a0000000-0000-4000-8000-000000000001";

app.use(helmet());
app.use(
  cors({
    origin: process.env.FRONTEND_URL || "http://localhost:3000",
    credentials: true,
  })
);
app.use(express.json());

const pool = process.env.DATABASE_URL
  ? new pg.Pool({ connectionString: process.env.DATABASE_URL })
  : null;

app.get("/health", (_req, res) => {
  res.json({ status: "ok", service: "api_gateway" });
});

app.get("/api/v1/tenants/:slug", async (req, res) => {
  if (!pool) {
    return res.status(503).json({ error: "Base de datos no configurada" });
  }
  try {
    const { rows } = await pool.query(
      `SELECT id, name, slug, legal_name, company_size, legal_nature
       FROM tenants WHERE slug = $1 AND is_active = TRUE`,
      [req.params.slug]
    );
    if (!rows.length) {
      return res.status(404).json({ error: "Tenant no encontrado" });
    }
    res.json(rows[0]);
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Error al consultar tenant" });
  }
});

app.get("/api/v1/tenants/:slug/dashboard", async (req, res) => {
  if (!pool) {
    return res.status(503).json({ error: "Base de datos no configurada" });
  }
  try {
    const tenantResult = await pool.query(
      `SELECT id, name, slug FROM tenants WHERE slug = $1`,
      [req.params.slug]
    );
    if (!tenantResult.rows.length) {
      return res.status(404).json({ error: "Tenant no encontrado" });
    }
    const tenantId = tenantResult.rows[0].id;

    const [assessments, tasks] = await Promise.all([
      pool.query(
        `SELECT id, assessment_type, status,
                score_governance, score_legal_bases, score_security,
                score_rights_mgmt, score_third_parties, created_at
         FROM compliance_assessments
         WHERE tenant_id = $1
         ORDER BY created_at DESC LIMIT 5`,
        [tenantId]
      ),
      pool.query(
        `SELECT id, title, priority, status, due_date, legal_reference
         FROM remediation_tasks
         WHERE tenant_id = $1 AND status NOT IN ('completed', 'cancelled')
         ORDER BY due_date ASC NULLS LAST LIMIT 10`,
        [tenantId]
      ),
    ]);

    const latest = assessments.rows[0];
    const scores = latest
      ? {
          governance: Number(latest.score_governance ?? 0),
          legal_bases: Number(latest.score_legal_bases ?? 0),
          security: Number(latest.score_security ?? 0),
          rights_mgmt: Number(latest.score_rights_mgmt ?? 0),
          third_parties: Number(latest.score_third_parties ?? 0),
        }
      : null;

    const globalScore = scores
      ? Math.round(
          scores.governance * 0.2 +
            scores.legal_bases * 0.25 +
            scores.security * 0.25 +
            scores.rights_mgmt * 0.15 +
            scores.third_parties * 0.15
        )
      : 0;

    res.json({
      tenant: tenantResult.rows[0],
      global_score: globalScore,
      compliance_level:
        globalScore >= 85
          ? "compliant"
          : globalScore >= 70
            ? "partial"
            : globalScore >= 50
              ? "non_compliant"
              : "high_risk",
      latest_assessment: latest,
      dimension_scores: scores,
      remediation_tasks: tasks.rows,
    });
  } catch (err) {
    console.error(err);
    res.status(500).json({ error: "Error al cargar dashboard" });
  }
});

app.use(
  "/api/v1/ai",
  createProxyMiddleware({
    target: AI_SERVICE_URL,
    changeOrigin: true,
    pathRewrite: { "^/api/v1/ai": "/v1" },
    on: {
      proxyReq: (proxyReq, req) => {
        if (!req.headers["x-tenant-id"]) {
          proxyReq.setHeader("X-Tenant-ID", DEMO_TENANT_ID);
        }
      },
    },
  })
);

app.listen(PORT, () => {
  console.log(`API Gateway escuchando en http://localhost:${PORT}`);
});
