import Link from "next/link";

export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-8 p-8">
      <div className="max-w-xl text-center">
        <p className="mb-2 text-sm font-medium uppercase tracking-wide text-blue-700">
          Ley 1581 · GDPR · ISO 27701
        </p>
        <h1 className="mb-4 text-4xl font-bold text-slate-900">
          Autodiagnóstico de Protección de Datos
        </h1>
        <p className="mb-8 text-lg text-slate-600">
          Evalúe el cumplimiento normativo de su organización con motor RAG
          jurídico y plan de remediación accionable.
        </p>
        <Link
          href="/app/empresa-demo/dashboard"
          className="inline-flex rounded-lg bg-blue-800 px-6 py-3 font-semibold text-white transition hover:bg-blue-900"
        >
          Entrar al demo — Empresa Demo SAS
        </Link>
      </div>
    </main>
  );
}
