import { ComplianceGauge } from "@/components/dashboard/ComplianceGauge";
import { DimensionRadarChart } from "@/components/dashboard/DimensionRadarChart";
import { RAGQueryPanel } from "@/components/dashboard/RAGQueryPanel";
import { fetchDashboard } from "@/lib/api-client";

export default async function DashboardPage({
  params,
}: {
  params: { "tenant-slug": string };
}) {
  const slug = params["tenant-slug"];
  let data;
  try {
    data = await fetchDashboard(slug);
  } catch {
    return (
      <p className="text-red-600">
        No se pudo conectar con el API. Ejecute{" "}
        <code className="rounded bg-slate-200 px-1">docker compose up</code>.
      </p>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-3">
        <ComplianceGauge score={data.global_score} level={data.compliance_level} />
        {data.dimension_scores && (
          <div className="lg:col-span-2">
            <DimensionRadarChart scores={data.dimension_scores} />
          </div>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <RAGQueryPanel />
        <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-slate-900">
            Tareas de remediación activas
          </h2>
          {data.remediation_tasks.length === 0 ? (
            <p className="text-sm text-slate-500">Sin tareas pendientes.</p>
          ) : (
            <ul className="space-y-3">
              {data.remediation_tasks.map((task) => (
                <li
                  key={task.id}
                  className="rounded-lg border border-slate-100 bg-slate-50 p-3 text-sm"
                >
                  <p className="font-medium text-slate-900">{task.title}</p>
                  <p className="mt-1 text-xs text-slate-500">{task.legal_reference}</p>
                  <div className="mt-2 flex gap-2 text-xs">
                    <span className="rounded bg-amber-100 px-2 py-0.5 text-amber-800">
                      {task.priority}
                    </span>
                    <span className="rounded bg-blue-100 px-2 py-0.5 text-blue-800">
                      {task.status}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
}
