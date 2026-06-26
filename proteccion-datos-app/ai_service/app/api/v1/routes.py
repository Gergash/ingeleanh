from uuid import UUID

from fastapi import APIRouter, Depends, Header, HTTPException
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.models.schemas import (
    AssessmentAnalyzeRequest,
    AssessmentAnalyzeResponse,
    RAGQueryRequest,
    RAGQueryResponse,
)
from app.services.rag.engine import analyze_assessment, execute_rag_query

router = APIRouter()


def _parse_tenant_id(x_tenant_id: str | None) -> UUID | None:
    if not x_tenant_id:
        return None
    try:
        return UUID(x_tenant_id)
    except ValueError as exc:
        raise HTTPException(status_code=400, detail="X-Tenant-ID inválido") from exc


@router.get("/health")
async def health() -> dict:
    return {"status": "ok", "service": "ai_service"}


@router.post("/rag/query", response_model=RAGQueryResponse)
async def rag_query(
    body: RAGQueryRequest,
    db: AsyncSession = Depends(get_db),
    x_tenant_id: str | None = Header(default=None),
) -> RAGQueryResponse:
    tenant_id = _parse_tenant_id(x_tenant_id)
    jurisdictions = list(body.jurisdictions)
    if body.include_global and "CO" not in jurisdictions:
        jurisdictions.append("CO")

    return await execute_rag_query(
        db,
        query=body.query,
        tenant_id=tenant_id,
        jurisdictions=jurisdictions,
        max_chunks=body.max_chunks,
    )


@router.post("/assessment/analyze", response_model=AssessmentAnalyzeResponse)
async def assessment_analyze(
    body: AssessmentAnalyzeRequest,
    db: AsyncSession = Depends(get_db),
    x_tenant_id: str | None = Header(default=None),
) -> AssessmentAnalyzeResponse:
    tenant_id = _parse_tenant_id(x_tenant_id)
    if not tenant_id:
        raise HTTPException(status_code=400, detail="X-Tenant-ID requerido")

    return await analyze_assessment(
        db,
        assessment_id=body.assessment_id,
        tenant_id=tenant_id,
    )
