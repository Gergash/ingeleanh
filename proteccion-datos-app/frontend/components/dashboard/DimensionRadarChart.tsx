type DimensionScores = {
  governance: number;
  legal_bases: number;
  security: number;
  rights_mgmt: number;
  third_parties: number;
};

const LABELS: Record<keyof DimensionScores, string> = {
  governance: "Gobernanza (20%)",
  legal_bases: "Bases jurídicas (25%)",
  security: "Seguridad (25%)",
  rights_mgmt: "Derechos titulares (15%)",
  third_parties: "Terceros (15%)",
};

export function DimensionRadarChart({ scores }: { scores: DimensionScores }) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
      <h2 className="mb-4 text-lg font-semibold text-slate-900">Dimensiones de evaluación</h2>
      <ul className="space-y-3">
        {(Object.keys(scores) as Array<keyof DimensionScores>).map((key) => (
          <li key={key}>
            <div className="mb-1 flex justify-between text-sm">
              <span className="text-slate-700">{LABELS[key]}</span>
              <span className="font-semibold text-slate-900">{scores[key]}</span>
            </div>
            <div className="h-2 overflow-hidden rounded-full bg-slate-100">
              <div
                className="h-full rounded-full bg-blue-700 transition-all"
                style={{ width: `${scores[key]}%` }}
              />
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
