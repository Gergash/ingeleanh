import { AppSidebar } from "@/components/shared/AppSidebar";

export default function TenantLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: { "tenant-slug": string };
}) {
  const slug = params["tenant-slug"];
  const displayName = slug.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

  return (
    <div className="flex min-h-screen">
      <AppSidebar tenantSlug={slug} tenantName={displayName} />
      <div className="flex flex-1 flex-col">
        <header className="border-b border-slate-200 bg-white px-6 py-4">
          <h1 className="text-lg font-semibold text-slate-900">Panel de cumplimiento</h1>
        </header>
        <main className="flex-1 bg-slate-50 p-6">{children}</main>
      </div>
    </div>
  );
}
