type Props = {
  score: number;
  level: string;
};

const LEVEL_LABELS: Record<string, string> = {
  compliant: "Conforme",
  partial: "Parcialmente conforme",
  non_compliant: "No conforme",
  high_risk: "Alto riesgo",
  critical: "Crítico",
};

const LEVEL_COLORS: Record<string, string> = {
  compliant: "text-emerald-700 bg-emerald-50",
  partial: "text-amber-700 bg-amber-50",
  non_compliant: "text-orange-700 bg-orange-50",
  high_risk: "text-red-700 bg-red-50",
  critical: "text-red-900 bg-red-100",
};

export function ComplianceGauge({ score, level }: Props) {
  const color = LEVEL_COLORS[level] ?? "text-slate-700 bg-slate-100";
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
      <p className="text-sm font-medium text-slate-500">Score global de cumplimiento</p>
      <p className="mt-2 text-5xl font-bold text-blue-900">{score}</p>
      <span className={`mt-3 inline-block rounded-full px-3 py-1 text-sm font-medium ${color}`}>
        {LEVEL_LABELS[level] ?? level}
      </span>
    </div>
  );
}
