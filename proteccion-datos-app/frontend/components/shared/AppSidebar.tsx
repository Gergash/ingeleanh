import Link from "next/link";

const NAV = [
  { href: "dashboard", label: "Dashboard" },
  { href: "assessments", label: "Diagnósticos" },
  { href: "remediation", label: "Remediación" },
];

export function AppSidebar({
  tenantSlug,
  tenantName,
}: {
  tenantSlug: string;
  tenantName: string;
}) {
  return (
    <aside className="flex w-64 flex-col border-r border-slate-200 bg-slate-900 text-white">
      <div className="border-b border-slate-700 p-5">
        <p className="text-xs uppercase tracking-wide text-slate-400">Tenant</p>
        <p className="mt-1 font-semibold">{tenantName}</p>
      </div>
      <nav className="flex-1 p-4">
        <ul className="space-y-1">
          {NAV.map((item) => (
            <li key={item.href}>
              <Link
                href={`/app/${tenantSlug}/${item.href}`}
                className="block rounded-lg px-3 py-2 text-sm text-slate-300 hover:bg-slate-800 hover:text-white"
              >
                {item.label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <div className="border-t border-slate-700 p-4 text-xs text-slate-500">
        Protección de Datos v0.1
      </div>
    </aside>
  );
}
