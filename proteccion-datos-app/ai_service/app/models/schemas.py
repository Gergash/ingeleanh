from uuid import UUID

from pydantic import BaseModel, Field


class Citation(BaseModel):
    source_law: str
    article_number: str | None = None
    excerpt: str


class ChunkReference(BaseModel):
    id: UUID
    source_law: str
    article_number: str | None = None
    rrf_score: float | None = None


class RAGQueryRequest(BaseModel):
    query: str = Field(..., min_length=5, max_length=2000)
    jurisdictions: list[str] = Field(default=["CO"])
    include_global: bool = True
    max_chunks: int = Field(default=5, ge=1, le=20)
    assessment_id: UUID | None = None


class RAGQueryResponse(BaseModel):
    answer: str
    confidence_score: float
    requires_human_review: bool
    citations: list[Citation]
    chunks_used: list[ChunkReference]
    processing_time_ms: int


class AssessmentAnalyzeRequest(BaseModel):
    assessment_id: UUID
    dimension: str | None = None


class AssessmentAnalyzeResponse(BaseModel):
    assessment_id: UUID
    summary: str
    gaps: list[dict]
    recommendations: list[dict]
    confidence_score: float
