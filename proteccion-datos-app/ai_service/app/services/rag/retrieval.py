from uuid import UUID

from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession


async def hybrid_retrieve(
    db: AsyncSession,
    *,
    query: str,
    tenant_id: UUID | None,
    jurisdictions: list[str],
    sensitivity_level: int = 5,
    limit: int = 20,
) -> list[dict]:
    """
    Búsqueda híbrida con Filtering Wall obligatorio (TDD §1.4).
    MVP: BM25 vía full-text search; búsqueda densa cuando existan embeddings.
    """
    tenant_param = str(tenant_id) if tenant_id else None

    sparse_sql = text("""
        SELECT
            dc.id,
            dc.content,
            dc.source_law,
            dc.article_number,
            dc.jurisdiction,
            ts_rank(dc.content_tsv, plainto_tsquery('spanish', :query)) AS bm25_score
        FROM document_chunks dc
        WHERE
            (dc.tenant_id = CAST(:tenant_id AS uuid) OR dc.tenant_id IS NULL)
            AND dc.jurisdiction = ANY(:jurisdictions)
            AND dc.effective_date <= CURRENT_DATE
            AND dc.sensitivity_level <= :sensitivity_level
            AND dc.content_tsv @@ plainto_tsquery('spanish', :query)
        ORDER BY bm25_score DESC
        LIMIT :limit
    """)

    sparse_result = await db.execute(
        sparse_sql,
        {
            "query": query,
            "tenant_id": tenant_param,
            "jurisdictions": jurisdictions,
            "sensitivity_level": sensitivity_level,
            "limit": limit,
        },
    )
    sparse_rows = [
        {
            "id": row.id,
            "content": row.content,
            "source_law": row.source_law,
            "article_number": row.article_number,
            "jurisdiction": row.jurisdiction,
            "bm25_score": float(row.bm25_score or 0),
        }
        for row in sparse_result.fetchall()
    ]

    if sparse_rows:
        return sparse_rows

    fallback_sql = text("""
        SELECT
            dc.id,
            dc.content,
            dc.source_law,
            dc.article_number,
            dc.jurisdiction,
            0.1 AS bm25_score
        FROM document_chunks dc
        WHERE
            (dc.tenant_id = CAST(:tenant_id AS uuid) OR dc.tenant_id IS NULL)
            AND dc.jurisdiction = ANY(:jurisdictions)
            AND dc.sensitivity_level <= :sensitivity_level
        ORDER BY dc.created_at DESC
        LIMIT :limit
    """)

    fallback_result = await db.execute(
        fallback_sql,
        {
            "tenant_id": tenant_param,
            "jurisdictions": jurisdictions,
            "sensitivity_level": sensitivity_level,
            "limit": limit,
        },
    )
    return [
        {
            "id": row.id,
            "content": row.content,
            "source_law": row.source_law,
            "article_number": row.article_number,
            "jurisdiction": row.jurisdiction,
            "bm25_score": float(row.bm25_score),
        }
        for row in fallback_result.fetchall()
    ]
