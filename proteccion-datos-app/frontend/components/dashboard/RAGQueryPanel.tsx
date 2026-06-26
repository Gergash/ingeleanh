"use client";

import { useState } from "react";
import { queryRAG, type RAGResponse } from "@/lib/api-client";

export function RAGQueryPanel() {
  const [query, setQuery] = useState(
    "¿Qué principios debe cumplir el tratamiento de datos personales?"
  );
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<RAGResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const data = await queryRAG(query);
      setResult(data);
    } catch {
      setError("No se pudo ejecutar la consulta. Verifique que el API esté activo.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
      <h2 className="mb-1 text-lg font-semibold text-slate-900">
        Consulta normativa (RAG)
      </h2>
      <p className="mb-4 text-sm text-slate-500">
        Motor híbrido con Filtering Wall multitenant — Ley 1581 / GDPR
      </p>
      <form onSubmit={handleSubmit} className="space-y-3">
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          rows={3}
          className="w-full rounded-lg border border-slate-300 p-3 text-sm focus:border-blue-600 focus:outline-none"
        />
        <button
          type="submit"
          disabled={loading}
          className="rounded-lg bg-emerald-700 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-800 disabled:opacity-50"
        >
          {loading ? "Consultando..." : "Consultar"}
        </button>
      </form>
      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
      {result && (
        <div className="mt-4 space-y-3 rounded-lg bg-slate-50 p-4 text-sm">
          <p className="text-slate-800">{result.answer}</p>
          <div className="flex flex-wrap gap-3 text-xs text-slate-600">
            <span>Confianza: {(result.confidence_score * 100).toFixed(0)}%</span>
            <span>{result.processing_time_ms} ms</span>
            {result.requires_human_review && (
              <span className="font-medium text-amber-700">Revisión DPO recomendada</span>
            )}
          </div>
          {result.citations.length > 0 && (
            <ul className="space-y-1 border-t border-slate-200 pt-3">
              {result.citations.map((c, i) => (
                <li key={i} className="text-xs text-slate-600">
                  <strong>{c.source_law}</strong>
                  {c.article_number && ` — Art. ${c.article_number}`}: {c.excerpt}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  );
}
