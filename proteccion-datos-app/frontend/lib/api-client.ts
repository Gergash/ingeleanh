const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:4000";

export type DashboardData = {
  tenant: { id: string; name: string; slug: string };
  global_score: number;
  compliance_level: string;
  dimension_scores: {
    governance: number;
    legal_bases: number;
    security: number;
    rights_mgmt: number;
    third_parties: number;
  } | null;
  remediation_tasks: Array<{
    id: string;
    title: string;
    priority: string;
    status: string;
    due_date: string | null;
    legal_reference: string | null;
  }>;
};

export type RAGResponse = {
  answer: string;
  confidence_score: number;
  requires_human_review: boolean;
  citations: Array<{
    source_law: string;
    article_number: string | null;
    excerpt: string;
  }>;
  processing_time_ms: number;
};

export async function fetchDashboard(tenantSlug: string): Promise<DashboardData> {
  const res = await fetch(`${API_URL}/api/v1/tenants/${tenantSlug}/dashboard`, {
    cache: "no-store",
  });
  if (!res.ok) throw new Error("No se pudo cargar el dashboard");
  return res.json();
}

export async function queryRAG(query: string): Promise<RAGResponse> {
  const res = await fetch(`${API_URL}/api/v1/ai/rag/query`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, jurisdictions: ["CO"], max_chunks: 5 }),
  });
  if (!res.ok) throw new Error("Error en consulta RAG");
  return res.json();
}
