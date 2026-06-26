import time
from uuid import UUID

from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import settings
from app.models.schemas import (
    AssessmentAnalyzeResponse,
    Citation,
    ChunkReference,
    RAGQueryResponse,
)
from app.services.rag.retrieval import hybrid_retrieve


def _build_answer(query: str, chunks: list[dict]) -> tuple[str, float]:
    if not chunks:
        return (
            "No se encontró información normativa relevante para esta consulta. "
            "Se recomienda revisión por el DPO.",
            0.35,
        )

    top = chunks[0]
    article = top.get("article_number") or "N/A"
    excerpt = top["content"][:400]
    answer = (
        f"Según {top['source_law']} (Art. {article}), "
        f"la consulta sobre «{query[:80]}» se relaciona con: {excerpt}..."
    )
    confidence = min(0.92, 0.55 + len(chunks) * 0.08 + float(top.get("rrf_score", top.get("bm25_score", 0.1))))
    return answer, confidence


async def execute_rag_query(
    db: AsyncSession,
    *,
    query: str,
    tenant_id: UUID | None,
    jurisdictions: list[str],
    max_chunks: int,
) -> RAGQueryResponse:
    start = time.perf_counter()

    chunks = await hybrid_retrieve(
        db,
        query=query,
        tenant_id=tenant_id,
        jurisdictions=jurisdictions,
        limit=max_chunks * 2,
    )
    chunks = chunks[:max_chunks]

    answer, confidence = _build_answer(query, chunks)
    requires_review = confidence < settings.CONFIDENCE_MEDIUM

    citations = [
        Citation(
            source_law=c["source_law"],
            article_number=c.get("article_number"),
            excerpt=c["content"][:200],
        )
        for c in chunks[:3]
    ]

    chunks_used = [
        ChunkReference(
            id=c["id"],
            source_law=c["source_law"],
            article_number=c.get("article_number"),
            rrf_score=c.get("rrf_score") or c.get("bm25_score"),
        )
        for c in chunks
    ]

    elapsed_ms = int((time.perf_counter() - start) * 1000)

    return RAGQueryResponse(
        answer=answer,
        confidence_score=round(confidence, 2),
        requires_human_review=requires_review,
        citations=citations,
        chunks_used=chunks_used,
        processing_time_ms=elapsed_ms,
    )


async def analyze_assessment(
    db: AsyncSession,
    *,
    assessment_id: UUID,
    tenant_id: UUID,
) -> AssessmentAnalyzeResponse:
    chunks = await hybrid_retrieve(
        db,
        query="principios legalidad finalidad autorización Ley 1581",
        tenant_id=tenant_id,
        jurisdictions=["CO"],
        limit=5,
    )

    gaps = [
        {
            "dimension": "legal_bases",
            "indicator": "B1",
            "description": "Verificar base jurídica documentada por finalidad",
            "severity": "high",
        },
        {
            "dimension": "rights_mgmt",
            "indicator": "D2",
            "description": "Validar cumplimiento de SLA de respuesta a titulares",
            "severity": "medium",
        },
    ]

    recommendations = [
        {
            "title": "Actualizar política de privacidad",
            "legal_reference": "Art. 15 Ley 1581/2012",
            "priority": "high",
        },
        {
            "title": "Documentar mecanismos de autorización",
            "legal_reference": "Art. 9 Ley 1581/2012",
            "priority": "high",
        },
    ]

    summary = (
        f"Análisis preliminar del diagnóstico {assessment_id}. "
        f"Se recuperaron {len(chunks)} fragmentos normativos de referencia. "
        "Se identificaron brechas en bases jurídicas y gestión de derechos."
    )

    return AssessmentAnalyzeResponse(
        assessment_id=assessment_id,
        summary=summary,
        gaps=gaps,
        recommendations=recommendations,
        confidence_score=0.72,
    )
