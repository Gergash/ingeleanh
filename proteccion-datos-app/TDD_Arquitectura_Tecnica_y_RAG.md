# TDD — Arquitectura Técnica y Motor RAG
## Plataforma SaaS de Autodiagnóstico de Protección de Datos

**Versión:** 1.0.0  
**Fecha:** 2026-06-26  
**Clasificación:** Confidencial — Uso Interno  
**Stack:** Next.js 14 · FastAPI · PostgreSQL + pgvector · LangGraph · Docker/Kubernetes  

---

## Tabla de Contenido

1. [Arquitectura RAG Agéntica y Base de Datos Vectorial](#1-arquitectura-rag-agéntica-y-base-de-datos-vectorial)
2. [Pipeline de Ingesta, Fragmentación y Reclasificación](#2-pipeline-de-ingesta-fragmentación-y-reclasificación)
3. [Esquemas Relacionales y de Seguridad](#3-esquemas-relacionales-y-de-seguridad)
4. [Stack Tecnológico Web](#4-stack-tecnológico-web)

---

## 1. Arquitectura RAG Agéntica y Base de Datos Vectorial

### 1.1 Visión General — Metadata-Reinforced Hybrid Retrieval

La arquitectura RAG del sistema no es un RAG naive. Implementa el patrón **Metadata-Reinforced Hybrid Retrieval** (MRHR), que combina cuatro mecanismos complementarios para garantizar recuperación de alta precisión con aislamiento absoluto multitenant.

```
ARQUITECTURA MRHR — FLUJO COMPLETO

[USUARIO / TENANT]
        │
        ▼
┌───────────────────────────────────────────────────────────────────┐
│  QUERY PROCESSOR                                                   │
│  ├── Detección de intención (legal query vs. factual vs. both)    │
│  ├── Expansión de query (sinónimos legales, artículos relacionados)│
│  └── Generación de vector de query (embedding model)              │
└───────────────────────────────────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────────────────────────────────┐
│  FILTERING WALL (PRE-BÚSQUEDA — OBLIGATORIO)                      │
│  ├── WHERE tenant_id = {tenant_id_del_jwt}                        │
│  ├── AND jurisdiction IN ({jurisdicciones_habilitadas})           │
│  ├── AND sensitivity_level <= {nivel_clearance_del_rol}           │
│  └── AND effective_date <= NOW()                                   │
│  ⚠ ESTE FILTRO SE APLICA ANTES DE LA BÚSQUEDA VECTORIAL          │
│    NUNCA SE OMITE — GARANTIZA AISLAMIENTO MULTITENANT             │
└───────────────────────────────────────────────────────────────────┘
        │
        ▼
┌──────────────────────────┬────────────────────────────────────────┐
│   BÚSQUEDA DENSA         │   BÚSQUEDA DISPERSA (BM25)             │
│   pgvector ANN           │   PostgreSQL full-text search          │
│   (similitud semántica)  │   (coincidencia léxica)                │
└──────────────────────────┴────────────────────────────────────────┘
        │                            │
        └────────────┬───────────────┘
                     ▼
        ┌─────────────────────────┐
        │  RECIPROCAL RANK FUSION │
        │  (RRF — k=60)           │
        │  Fusiona rankings de    │
        │  ambas búsquedas        │
        └─────────────────────────┘
                     │
                     ▼
        ┌─────────────────────────┐
        │  CROSS-ENCODER          │
        │  RERANKING              │
        │  Top-K → Top-N          │
        │  (mayor precisión)      │
        └─────────────────────────┘
                     │
                     ▼
┌───────────────────────────────────────────────────────────────────┐
│  LANGGRAPH AGENTIC ORCHESTRATOR                                    │
│  ├── Agent: Legal Reasoning (Ley 1581, GDPR, ISO 27701)           │
│  ├── Agent: Evidence Evaluator (análisis de evidencias del tenant) │
│  ├── Agent: Gap Analyzer (brechas entre estado real y normativa)   │
│  ├── Agent: Recommendation Generator (plan de remediación)        │
│  └── Agent: Citation Verifier (verifica citas normativas exactas)  │
└───────────────────────────────────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────────────────────────────────┐
│  RESPONSE GENERATOR + CONFIDENCE SCORING                           │
│  ├── Si confianza > 0.85: respuesta directa con citas             │
│  ├── Si confianza 0.60-0.85: respuesta con advertencia de revisar  │
│  └── Si confianza < 0.60: escalación a revisión humana (DPO)      │
└───────────────────────────────────────────────────────────────────┘
```

---

### 1.2 PostgreSQL + pgvector — Configuración de Producción

#### 1.2.1 Instalación y Configuración

```sql
-- Habilitar la extensión pgvector
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- Para BM25 híbrido
CREATE EXTENSION IF NOT EXISTS unaccent;      -- Para búsqueda sin tildes
CREATE EXTENSION IF NOT EXISTS pgcrypto;      -- Para cifrado de columnas
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";   -- Para generación de UUIDs

-- Verificar versión de pgvector (mínimo 0.6.0 para HNSW)
SELECT extversion FROM pg_extension WHERE extname = 'vector';
```

#### 1.2.2 Dimensiones de Vectores

| Modelo de Embedding | Dimensiones | Tokens Máximos | Uso Recomendado |
|---|---|---|---|
| OpenAI text-embedding-ada-002 | 1536 | 8191 | Producción — alto rendimiento |
| OpenAI text-embedding-3-small | 1536 | 8191 | Producción — coste reducido |
| OpenAI text-embedding-3-large | 3072 | 8191 | Máxima calidad — alto coste |
| sentence-transformers/paraphrase-multilingual-mpnet | 768 | 512 | Open-source — multilingüe |
| BAAI/bge-m3 | 1024 | 8192 | Open-source — mejor opción |
| intfloat/multilingual-e5-large | 1024 | 512 | Open-source — buena calidad |

**Configuración recomendada para producción:**

```sql
-- Tabla de embeddings con dimensión parametrizable
-- Para OpenAI ada-002 o 3-small:
CREATE TABLE document_embeddings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id        UUID NOT NULL REFERENCES document_chunks(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    embedding       vector(1536),    -- Para ada-002 / 3-small
    model_version   VARCHAR(64) NOT NULL DEFAULT 'text-embedding-ada-002',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_chunk_tenant CHECK (chunk_id IS NOT NULL AND tenant_id IS NOT NULL)
);

-- Si se usa modelo open-source con 768 dimensiones:
-- embedding vector(768)

-- Si se usa text-embedding-3-large:
-- embedding vector(3072)
```

---

### 1.3 Tipos de Índice — IVFFlat vs HNSW

#### 1.3.1 IVFFlat (Inverted File with Flat Quantization)

```sql
-- IVFFlat — División en listas (clusters de Voronoi)
-- CUÁNDO USAR: dataset mediano (100K - 10M vectores),
-- actualizaciones frecuentes de vectores, memoria limitada

CREATE INDEX CONCURRENTLY idx_embeddings_ivfflat
ON document_embeddings
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 200);  -- Regla: lists ≈ sqrt(num_rows)
                     -- Para 1M de filas: lists = 1000
                     -- Para 40K de filas: lists = 200

-- Configurar probes para búsqueda (mayor = más preciso, más lento)
-- Para producción: probes = lists / 10
SET ivfflat.probes = 20;

-- LIMITACIONES IMPORTANTES:
-- 1. Requiere VACUUM después de muchas inserciones/eliminaciones
-- 2. La precisión (recall) depende de probes — ajustar por benchmarking
-- 3. No soporta vectores NULL (usar COALESCE o filtrar NULLs)
```

#### 1.3.2 HNSW (Hierarchical Navigable Small World)

```sql
-- HNSW — Grafo de proximidad jerárquico
-- CUÁNDO USAR: dataset grande (>1M vectores) o cuando la latencia
-- de búsqueda es crítica, actualizaciones poco frecuentes,
-- suficiente RAM para mantener el grafo en memoria

CREATE INDEX CONCURRENTLY idx_embeddings_hnsw
ON document_embeddings
USING hnsw (embedding vector_cosine_ops)
WITH (
    m = 16,           -- Número de conexiones por nodo (16-64)
                      -- Mayor m = mayor recall, mayor memoria
    ef_construction = 64  -- Tamaño de la lista candidata durante construcción
                          -- Mayor = mejor calidad, más lento en indexación
);

-- Para búsqueda en producción — ajustar ef_search
SET hnsw.ef_search = 40;  -- Mayor = mejor recall, mayor latencia

-- VENTAJAS vs IVFFlat:
-- - ~10-100x más rápido en búsqueda aproximada
-- - No requiere etapa de entrenamiento (como IVFFlat)
-- - Mejor recall para el mismo tiempo de consulta
-- DESVENTAJAS:
-- - Mayor consumo de memoria (grafo en RAM)
-- - Construcción del índice más lenta
-- - No soporta eliminación eficiente de vectores

-- DECISIÓN PARA ESTE SISTEMA:
-- Usar HNSW para documentos legales (corpus estático, alta demanda de velocidad)
-- Usar IVFFlat para evidencias de tenants (actualizaciones frecuentes)
```

---

### 1.4 Filtrado Pre-Búsqueda — Aislamiento Multitenant Absoluto

El **Filtering Wall** es el mecanismo de seguridad más crítico del sistema RAG. Garantiza que NINGÚN tenant pueda ver datos vectoriales de otro tenant.

```sql
-- CONSULTA RAG CON FILTERING WALL — PATRÓN OBLIGATORIO
-- Este patrón NUNCA se omite, incluso para superadmin

-- Paso 1: Búsqueda vectorial con pre-filtro obligatorio
WITH filtered_chunks AS (
    SELECT 
        dc.id,
        dc.content,
        dc.metadata,
        dc.source_law,
        dc.article_number,
        dc.jurisdiction,
        de.embedding <=> $1::vector AS cosine_distance,  -- $1 = query embedding
        ts_rank(dc.content_tsv, plainto_tsquery('spanish', $2)) AS bm25_score  -- $2 = query text
    FROM document_chunks dc
    JOIN document_embeddings de ON de.chunk_id = dc.id
    WHERE 
        -- FILTERING WALL — SIEMPRE PRIMERO
        (dc.tenant_id = $3 OR dc.tenant_id IS NULL)  -- $3 = tenant_id del JWT
        AND dc.jurisdiction = ANY($4::text[])          -- $4 = jurisdicciones habilitadas
        AND dc.effective_date <= CURRENT_DATE
        AND dc.sensitivity_level <= $5                 -- $5 = nivel de clearance del rol
        -- Límite inicial amplio para RRF posterior
    ORDER BY cosine_distance ASC
    LIMIT 100  -- Candidatos para RRF
),
-- Paso 2: BM25 (búsqueda léxica) con mismo filtro
bm25_candidates AS (
    SELECT 
        dc.id,
        dc.content,
        dc.metadata,
        ts_rank(dc.content_tsv, plainto_tsquery('spanish', $2)) AS bm25_score,
        ROW_NUMBER() OVER (ORDER BY ts_rank(dc.content_tsv, plainto_tsquery('spanish', $2)) DESC) AS bm25_rank
    FROM document_chunks dc
    WHERE 
        (dc.tenant_id = $3 OR dc.tenant_id IS NULL)
        AND dc.jurisdiction = ANY($4::text[])
        AND dc.effective_date <= CURRENT_DATE
        AND dc.sensitivity_level <= $5
        AND dc.content_tsv @@ plainto_tsquery('spanish', $2)
    LIMIT 100
),
-- Paso 3: Reciprocal Rank Fusion (RRF)
dense_ranked AS (
    SELECT id, content, metadata, 
           ROW_NUMBER() OVER (ORDER BY cosine_distance ASC) AS dense_rank
    FROM filtered_chunks
),
rrf_scores AS (
    SELECT 
        COALESCE(d.id, b.id) AS id,
        COALESCE(d.content, b.content) AS content,
        COALESCE(d.metadata, b.metadata) AS metadata,
        -- RRF formula: 1/(k + rank), k=60 es estándar
        COALESCE(1.0 / (60.0 + d.dense_rank), 0) +
        COALESCE(1.0 / (60.0 + b.bm25_rank), 0) AS rrf_score
    FROM dense_ranked d
    FULL OUTER JOIN bm25_candidates b ON d.id = b.id
)
SELECT id, content, metadata, rrf_score
FROM rrf_scores
ORDER BY rrf_score DESC
LIMIT 20;  -- Top-20 para re-ranking con Cross-Encoder
```

---

### 1.5 Orchestración Agéntica con LangGraph

#### 1.5.1 Grafo de Agentes para Razonamiento Legal

```python
from langgraph.graph import StateGraph, END
from typing import TypedDict, Annotated, List
import operator

class LegalAssessmentState(TypedDict):
    tenant_id: str
    query: str
    query_embedding: List[float]
    retrieved_chunks: List[dict]
    reranked_chunks: List[dict]
    legal_analysis: str
    evidence_analysis: str
    gaps_identified: List[dict]
    recommendations: List[dict]
    citations: List[dict]
    confidence_score: float
    requires_human_review: bool

# Definición del grafo de agentes
workflow = StateGraph(LegalAssessmentState)

# Nodos del grafo
workflow.add_node("query_processor", process_query)
workflow.add_node("retrieval_wall", apply_filtering_wall)  # NUNCA omitir
workflow.add_node("hybrid_retriever", execute_hybrid_retrieval)
workflow.add_node("reranker", cross_encoder_rerank)
workflow.add_node("legal_analyzer", analyze_legal_requirements)
workflow.add_node("evidence_evaluator", evaluate_tenant_evidence)
workflow.add_node("gap_analyzer", identify_compliance_gaps)
workflow.add_node("recommendation_generator", generate_remediation_plan)
workflow.add_node("citation_verifier", verify_legal_citations)
workflow.add_node("confidence_scorer", compute_confidence)
workflow.add_node("human_escalation", escalate_to_human_review)
workflow.add_node("response_generator", generate_final_response)

# Edges (flujo del grafo)
workflow.set_entry_point("query_processor")
workflow.add_edge("query_processor", "retrieval_wall")
workflow.add_edge("retrieval_wall", "hybrid_retriever")
workflow.add_edge("hybrid_retriever", "reranker")
workflow.add_edge("reranker", "legal_analyzer")
workflow.add_edge("legal_analyzer", "evidence_evaluator")
workflow.add_edge("evidence_evaluator", "gap_analyzer")
workflow.add_edge("gap_analyzer", "recommendation_generator")
workflow.add_edge("recommendation_generator", "citation_verifier")
workflow.add_edge("citation_verifier", "confidence_scorer")

# Edge condicional — escalar a humano si confianza baja
workflow.add_conditional_edges(
    "confidence_scorer",
    lambda state: "human_escalation" if state["confidence_score"] < 0.60 
                  else "response_generator",
    {
        "human_escalation": "human_escalation",
        "response_generator": "response_generator"
    }
)

workflow.add_edge("human_escalation", END)
workflow.add_edge("response_generator", END)

app = workflow.compile()
```

#### 1.5.2 Estrategias de Fallback y Umbrales de Confianza

```python
CONFIDENCE_THRESHOLDS = {
    "high_confidence": 0.85,      # Respuesta directa con citas
    "medium_confidence": 0.60,    # Respuesta con advertencia
    "low_confidence": 0.40,       # Escalación a DPO
    "insufficient": 0.40          # Indicar que no hay suficiente información
}

FALLBACK_STRATEGIES = [
    {
        "trigger": "no_chunks_retrieved",
        "action": "expand_query_with_synonyms",
        "description": "Si el Filtering Wall no retorna resultados, expandir query"
    },
    {
        "trigger": "low_reranker_score",
        "action": "increase_retrieval_k",
        "description": "Aumentar k de 20 a 50 y re-aplicar reranking"
    },
    {
        "trigger": "citation_verification_failed",
        "action": "fallback_to_structured_rules",
        "description": "Usar reglas jurídicas deterministas en lugar de RAG"
    },
    {
        "trigger": "llm_timeout",
        "action": "return_cached_analysis",
        "description": "Retornar análisis cacheado si existe para query similar"
    }
]
```

---

## 2. Pipeline de Ingesta, Fragmentación y Reclasificación

### 2.1 Pipeline de Ingesta de Documentos Legales

El pipeline de ingesta transforma documentos normativos crudos (PDFs, DOCXs, HTMLs de Diarios Oficiales) en chunks vectorizados listos para búsqueda.

#### 2.1.1 Diagrama del Pipeline

```
FUENTES DE INGESTA
├── Ley 1581/2012 — PDF oficial Diario Oficial
├── Decreto 1377/2013 — PDF oficial
├── Decreto 090/2018 — PDF oficial RNBD
├── Circulares SIC — HTML scraping periódico
├── GDPR 2016/679 — HTML EUR-Lex (multilingüe)
├── ISO 27701:2019 — PDF comprado (licencia)
├── Decisiones SIC — PDF de resoluciones
└── Jurisprudencia relevante — PDF sentencias
        │
        ▼
[EXTRACTOR DE TEXTO]
├── PDF: PyMuPDF (fitz) — preserva estructura mejor que pypdf
├── DOCX: python-docx — extrae párrafos con estructura
├── HTML: BeautifulSoup + regex de limpieza
└── Validación: detección de PDF escaneado (OCR con Tesseract si necesario)
        │
        ▼
[PRE-PROCESAMIENTO]
├── Normalización de encoding (UTF-8)
├── Eliminación de headers/footers repetitivos
├── Detección y preservación de estructura (artículos, parágrafos, numerales)
├── Normalización de referencias cruzadas ("conforme al artículo X")
└── Detección de idioma (español/inglés para GDPR)
        │
        ▼
[FRAGMENTADOR INTELIGENTE]
├── Respeta integridad de artículos (NUNCA parte un artículo)
├── 512-1024 tokens por chunk (con solapamiento configurable)
├── Overlap: 128 tokens (captura contexto entre chunks adyacentes)
└── Metadata extraction automática
        │
        ▼
[ENRIQUECIMIENTO DE METADATA]
├── source_law, article_number, jurisdiction
├── effective_date, expiry_date (si aplica)
├── sensitivity_level, tenant_id
└── content_hash (SHA-256 para deduplicación)
        │
        ▼
[MODELO DE EMBEDDING]
├── Generación de vector de embedding por chunk
├── Batch processing (no llamar API por chunk individual)
└── Almacenamiento en PostgreSQL + pgvector
        │
        ▼
[ÍNDICE BM25]
├── Actualización de tsvector para búsqueda léxica
└── Actualización de índices GiST/GIN
```

#### 2.1.2 Extractor y Fragmentador — Implementación

```python
import fitz  # PyMuPDF
import re
from dataclasses import dataclass
from typing import List, Optional

@dataclass
class LegalChunk:
    content: str
    source_law: str
    article_number: Optional[str]
    paragraph_number: Optional[str]
    jurisdiction: str
    effective_date: str
    tenant_id: Optional[str]  # None para documentos del corpus global
    sensitivity_level: int    # 1-5
    content_hash: str
    token_count: int

class LegalDocumentChunker:
    """
    Fragmentador especializado para textos legales.
    PRINCIPIO FUNDAMENTAL: Nunca partir un artículo a la mitad.
    """
    
    ARTICLE_PATTERNS = [
        r'(?i)^(art[íi]culo\s+\d+[\w°]*\.?)',   # "Artículo 5."
        r'(?i)^(art\.\s*\d+[\w°]*\.?)',           # "Art. 5."
        r'(?i)^(article\s+\d+)',                   # GDPR en inglés
        r'(?i)^(\d+\.\s)',                         # "5. Texto del artículo"
    ]
    
    def __init__(
        self,
        target_tokens: int = 512,
        max_tokens: int = 1024,
        overlap_tokens: int = 128,
        tokenizer_model: str = "cl100k_base"  # tiktoken para OpenAI
    ):
        self.target_tokens = target_tokens
        self.max_tokens = max_tokens
        self.overlap_tokens = overlap_tokens
        import tiktoken
        self.tokenizer = tiktoken.get_encoding(tokenizer_model)
    
    def extract_from_pdf(self, pdf_path: str) -> List[str]:
        """Extrae texto preservando estructura de artículos."""
        doc = fitz.open(pdf_path)
        articles = []
        current_article = []
        
        for page in doc:
            blocks = page.get_text("blocks")
            for block in blocks:
                text = block[4].strip()
                if self._is_article_start(text):
                    if current_article:
                        articles.append("\n".join(current_article))
                    current_article = [text]
                elif current_article:
                    current_article.append(text)
        
        if current_article:
            articles.append("\n".join(current_article))
        
        return articles
    
    def _is_article_start(self, text: str) -> bool:
        return any(re.match(p, text.strip()) for p in self.ARTICLE_PATTERNS)
    
    def chunk_article(self, article_text: str, metadata: dict) -> List[LegalChunk]:
        """
        Fragmenta un artículo respetando su integridad.
        Si el artículo cabe en max_tokens, es un solo chunk.
        Si no, se divide por parágrafos con overlap.
        """
        tokens = self.tokenizer.encode(article_text)
        
        if len(tokens) <= self.max_tokens:
            # El artículo completo cabe — preservar integridad
            return [self._create_chunk(article_text, metadata)]
        
        # Dividir por parágrafos con overlap
        paragraphs = article_text.split("\n\n")
        chunks = []
        current_chunk_tokens = []
        current_chunk_text = []
        
        for paragraph in paragraphs:
            para_tokens = self.tokenizer.encode(paragraph)
            
            if len(current_chunk_tokens) + len(para_tokens) > self.target_tokens:
                if current_chunk_text:
                    chunks.append(self._create_chunk(
                        "\n\n".join(current_chunk_text), metadata
                    ))
                    # Overlap: mantener últimos overlap_tokens del chunk anterior
                    overlap_text = self._get_overlap_text(current_chunk_text)
                    current_chunk_text = [overlap_text, paragraph]
                    current_chunk_tokens = self.tokenizer.encode(
                        "\n\n".join(current_chunk_text)
                    )
                else:
                    # Párrafo único que excede el límite — split por oraciones
                    current_chunk_text = [paragraph]
                    current_chunk_tokens = para_tokens
            else:
                current_chunk_text.append(paragraph)
                current_chunk_tokens.extend(para_tokens)
        
        if current_chunk_text:
            chunks.append(self._create_chunk(
                "\n\n".join(current_chunk_text), metadata
            ))
        
        return chunks
    
    def _create_chunk(self, content: str, metadata: dict) -> LegalChunk:
        import hashlib
        return LegalChunk(
            content=content,
            source_law=metadata["source_law"],
            article_number=metadata.get("article_number"),
            paragraph_number=metadata.get("paragraph_number"),
            jurisdiction=metadata["jurisdiction"],
            effective_date=metadata["effective_date"],
            tenant_id=metadata.get("tenant_id"),
            sensitivity_level=metadata.get("sensitivity_level", 1),
            content_hash=hashlib.sha256(content.encode()).hexdigest(),
            token_count=len(self.tokenizer.encode(content))
        )
    
    def _get_overlap_text(self, text_parts: List[str]) -> str:
        """Retorna el texto de overlap del chunk anterior."""
        full_text = "\n\n".join(text_parts)
        tokens = self.tokenizer.encode(full_text)
        overlap_tokens = tokens[-self.overlap_tokens:]
        return self.tokenizer.decode(overlap_tokens)
```

---

### 2.2 Schema de Metadata por Chunk

```python
CHUNK_METADATA_SCHEMA = {
    # IDENTIFICACIÓN
    "chunk_id": "UUID — generado automáticamente",
    "content_hash": "SHA-256 del contenido — para deduplicación",
    
    # FUENTE LEGAL
    "source_law": {
        "type": "string",
        "enum": [
            "LEY_1581_2012",           # Ley 1581 de 2012 — Colombia
            "DECRETO_1377_2013",        # Decreto reglamentario 1377/2013
            "DECRETO_090_2018",         # Decreto RNBD 090/2018
            "CIRCULAR_SIC_2021",        # Circulares SIC
            "GDPR_2016_679",           # GDPR UE
            "ISO_27701_2019",          # ISO/IEC 27701:2019
            "ISO_27001_2022",          # ISO/IEC 27001:2022
            "EDPB_GUIDELINE",          # Guidelines del EDPB
            "SIC_RESOLUTION",          # Resoluciones SIC
            "CCONST_COLOMBIA",         # Corte Constitucional Colombia
            "TENANT_DOCUMENT"          # Documento propio del tenant (privado)
        ]
    },
    
    "article_number": "string | null — '5', '9(2)(a)', 'Art. 15'",
    "paragraph_number": "string | null — 'parágrafo 1', 'considerando 38'",
    "section_title": "string | null — título de la sección",
    
    # JURISDICCIÓN Y VIGENCIA
    "jurisdiction": {
        "type": "string",
        "enum": ["CO", "EU", "GLOBAL", "UK"]  # Colombia, UE, Global, UK post-Brexit
    },
    "effective_date": "date — fecha desde la que está vigente",
    "expiry_date": "date | null — fecha de derogación (null = vigente)",
    "version": "string — '1.0', '2.0' para documentos que se actualizan",
    
    # AISLAMIENTO MULTITENANT
    "tenant_id": "UUID | null — null para corpus público/global",
    "sensitivity_level": {
        "type": "integer",
        "range": "1-5",
        "values": {
            "1": "Público — todos los usuarios pueden ver",
            "2": "Interno — solo usuarios autenticados del tenant",
            "3": "Confidencial — DPO y Tenant Admin",
            "4": "Restringido — solo DPO",
            "5": "Ultra-restringido — solo Superadmin"
        }
    },
    
    # INDEXACIÓN Y CALIDAD
    "language": "string — 'es' | 'en'",
    "token_count": "integer — número de tokens del chunk",
    "embedding_model": "string — modelo usado para generar el embedding",
    "ingestion_date": "timestamp — cuándo se ingirió el documento",
    "last_verified_date": "timestamp — última verificación de vigencia",
    "quality_score": "float 0-1 — calidad de extracción (OCR confidence, etc.)"
}
```

---

### 2.3 Modelos de Embedding — Comparativa y Selección

| Modelo | Dims | Contexto | Calidad (MTEB) | Coste | Privacidad | Recomendación |
|---|---|---|---|---|---|---|
| text-embedding-ada-002 | 1536 | 8191 | Buena | $0.0001/1K tokens | Datos van a OpenAI | Producción estándar |
| text-embedding-3-small | 1536 | 8191 | Mejor que ada-002 | $0.00002/1K tokens | Datos van a OpenAI | Producción — relación coste/calidad |
| text-embedding-3-large | 3072 | 8191 | Excelente | $0.00013/1K tokens | Datos van a OpenAI | Cuando la calidad es crítica |
| BAAI/bge-m3 | 1024 | 8192 | Muy buena | Gratuito (self-hosted) | Datos en servidor propio | Self-hosted — mejor opción OSS |
| multilingual-e5-large | 1024 | 512 | Buena | Gratuito (self-hosted) | Datos en servidor propio | Self-hosted — contexto limitado |

**Decisión de Arquitectura:**

Para este sistema, se implementa una estrategia dual:
- **Corpus legal global** (Ley 1581, GDPR, ISO 27701): `text-embedding-3-small` — los documentos son públicos, el coste es mínimo en corpus fijo
- **Documentos del tenant** (políticas propias, DPAs, evidencias): `BAAI/bge-m3` self-hosted — los documentos del tenant son confidenciales y NO deben enviarse a APIs externas

```python
class EmbeddingRouter:
    """Selecciona el modelo de embedding según la sensibilidad del documento."""
    
    def __init__(self):
        from openai import OpenAI
        from sentence_transformers import SentenceTransformer
        
        self.openai_client = OpenAI()
        self.local_model = SentenceTransformer("BAAI/bge-m3")
    
    def embed(self, text: str, tenant_id: str | None) -> list[float]:
        if tenant_id is None:
            # Documento público del corpus legal — OpenAI OK
            response = self.openai_client.embeddings.create(
                input=text,
                model="text-embedding-3-small"
            )
            return response.data[0].embedding
        else:
            # Documento privado del tenant — modelo local
            return self.local_model.encode(text).tolist()
```

---

### 2.4 Reciprocal Rank Fusion (RRF)

El algoritmo RRF combina los rankings de búsqueda densa y dispersa sin necesitar puntajes normalizados.

```python
def reciprocal_rank_fusion(
    dense_results: list[dict],
    sparse_results: list[dict],
    k: int = 60
) -> list[dict]:
    """
    Implementación de RRF (Cormack et al., 2009).
    
    Args:
        dense_results: Lista de chunks ordenados por similitud vectorial
        sparse_results: Lista de chunks ordenados por BM25
        k: Constante de RRF (60 es estándar de la literatura)
    
    Returns:
        Lista de chunks ordenados por score RRF combinado
    """
    scores: dict[str, float] = {}
    chunks: dict[str, dict] = {}
    
    # Procesar resultados de búsqueda densa
    for rank, chunk in enumerate(dense_results, start=1):
        chunk_id = chunk["id"]
        scores[chunk_id] = scores.get(chunk_id, 0) + 1.0 / (k + rank)
        chunks[chunk_id] = chunk
    
    # Procesar resultados de búsqueda dispersa (BM25)
    for rank, chunk in enumerate(sparse_results, start=1):
        chunk_id = chunk["id"]
        scores[chunk_id] = scores.get(chunk_id, 0) + 1.0 / (k + rank)
        chunks[chunk_id] = chunk
    
    # Ordenar por score RRF descendente
    sorted_ids = sorted(scores.keys(), key=lambda x: scores[x], reverse=True)
    
    return [
        {**chunks[chunk_id], "rrf_score": scores[chunk_id]}
        for chunk_id in sorted_ids
    ]
```

---

### 2.5 Cross-Encoder Reranking

Después de RRF, los top-20 candidatos se rerankean con un Cross-Encoder para maximizar la precisión semántica.

```python
from sentence_transformers import CrossEncoder
import numpy as np

class LegalCrossEncoderReranker:
    """
    Reranking con Cross-Encoder para precisión semántica superior.
    
    El Cross-Encoder analiza la query y el chunk juntos, capturando
    interacciones semánticas que los bi-encoders no pueden.
    """
    
    def __init__(self, model_name: str = "cross-encoder/ms-marco-MiniLM-L-6-v2"):
        # Para español jurídico, considerar fine-tuning sobre corpus legal
        self.model = CrossEncoder(model_name, max_length=512)
    
    def rerank(
        self,
        query: str,
        candidates: list[dict],
        top_n: int = 5
    ) -> list[dict]:
        """
        Rerankea candidatos usando el Cross-Encoder.
        
        Args:
            query: La consulta legal original
            candidates: Top-K chunks del RRF
            top_n: Cuántos chunks retornar después del reranking
        
        Returns:
            Top-N chunks rerankeados por relevancia semántica
        """
        if not candidates:
            return []
        
        # Crear pares (query, chunk_content) para el Cross-Encoder
        pairs = [(query, candidate["content"]) for candidate in candidates]
        
        # Puntuar todos los pares en un solo batch
        scores = self.model.predict(pairs, show_progress_bar=False)
        
        # Adjuntar scores y ordenar
        for candidate, score in zip(candidates, scores):
            candidate["reranker_score"] = float(score)
        
        reranked = sorted(candidates, key=lambda x: x["reranker_score"], reverse=True)
        
        # Filtrar por umbral de relevancia mínima
        MIN_RELEVANCE_SCORE = -3.0  # Cross-Encoder usa logits sin normalizar
        filtered = [c for c in reranked[:top_n] if c["reranker_score"] > MIN_RELEVANCE_SCORE]
        
        return filtered
```

---

### 2.6 Control de Versiones para Regulaciones Actualizadas

```sql
-- Tabla de versiones de documentos legales
CREATE TABLE legal_document_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES legal_documents(id),
    version_number  INTEGER NOT NULL,
    version_label   VARCHAR(32) NOT NULL,  -- "v1.0", "2024-amendment"
    effective_date  DATE NOT NULL,
    expiry_date     DATE,
    change_summary  TEXT NOT NULL,
    ingested_by     UUID REFERENCES users(id),
    ingested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_current      BOOLEAN NOT NULL DEFAULT FALSE,
    
    UNIQUE (document_id, version_number)
);

-- Proceso de re-ingesta de regulación actualizada:
-- 1. Insertar nueva versión con is_current = FALSE
-- 2. Re-fragmentar y re-vectorizar el nuevo documento
-- 3. Marcar chunks de la versión anterior como expired
-- 4. En transacción atómica: set is_current = FALSE en versión anterior,
--    set is_current = TRUE en nueva versión
-- 5. Regenerar índices (REINDEX CONCURRENTLY)

-- Procedimiento de migración de versión
CREATE OR REPLACE PROCEDURE migrate_document_version(
    p_document_id UUID,
    p_new_version_number INTEGER,
    p_effective_date DATE
)
LANGUAGE plpgsql
AS $$
BEGIN
    -- Desactivar versión actual
    UPDATE legal_document_versions
    SET is_current = FALSE,
        expiry_date = p_effective_date - INTERVAL '1 day'
    WHERE document_id = p_document_id
    AND is_current = TRUE;
    
    -- Activar nueva versión
    UPDATE legal_document_versions
    SET is_current = TRUE
    WHERE document_id = p_document_id
    AND version_number = p_new_version_number;
    
    -- Marcar chunks de versión anterior como inactivos
    UPDATE document_chunks
    SET effective_until = p_effective_date - INTERVAL '1 day'
    WHERE document_id = p_document_id
    AND version_number < p_new_version_number;
    
    COMMIT;
END;
$$;
```

---

## 3. Esquemas Relacionales y de Seguridad

### 3.1 Esquema Completo de PostgreSQL

#### 3.1.1 Tenants y Usuarios

```sql
-- ============================================================
-- TENANTS
-- ============================================================
CREATE TABLE tenants (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    VARCHAR(256) NOT NULL,
    slug                    VARCHAR(128) NOT NULL UNIQUE,
    nit                     VARCHAR(20),            -- NIT Colombia
    legal_name              VARCHAR(512) NOT NULL,
    country                 CHAR(2) NOT NULL DEFAULT 'CO',  -- ISO 3166-1 alpha-2
    industry_sector         VARCHAR(64),
    company_size            VARCHAR(16) CHECK (company_size IN (
                                'startup', 'small', 'medium', 'large', 'enterprise'
                            )),
    legal_nature            VARCHAR(32) CHECK (legal_nature IN (
                                'public_national', 'public_territorial',
                                'private_regulated', 'private_large',
                                'private_medium', 'private_small',
                                'ngo', 'startup'
                            )),
    subscription_plan       VARCHAR(32) NOT NULL DEFAULT 'trial',
    subscription_expires_at TIMESTAMPTZ,
    data_residency_region   VARCHAR(32) NOT NULL DEFAULT 'sa-east-1',
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,           -- Soft delete
    
    -- Metadatos del plan de suscripción
    max_assessments_per_month INTEGER NOT NULL DEFAULT 3,
    max_users               INTEGER NOT NULL DEFAULT 5,
    max_document_storage_mb INTEGER NOT NULL DEFAULT 500
);

CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_active ON tenants(is_active) WHERE deleted_at IS NULL;

-- ============================================================
-- USUARIOS
-- ============================================================
CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    email               VARCHAR(320) NOT NULL,
    -- NUNCA almacenar contraseñas en texto plano
    -- La autenticación es via OAuth2 + PKCE con proveedor externo
    -- o via magic links
    display_name        VARCHAR(256),
    given_name          VARCHAR(128),
    family_name         VARCHAR(128),
    phone_encrypted     BYTEA,  -- AES-256-GCM cifrado
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    is_email_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_totp_secret     BYTEA,  -- Secreto TOTP cifrado con AES-256
    last_login_at       TIMESTAMPTZ,
    last_login_ip       INET,   -- PostgreSQL tipo nativo para IPs
    failed_login_count  INTEGER NOT NULL DEFAULT 0,
    locked_until        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    
    UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;

-- ============================================================
-- ROLES Y PERMISOS (RBAC)
-- ============================================================
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),  -- NULL = rol global de plataforma
    name        VARCHAR(64) NOT NULL,
    description TEXT,
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,  -- TRUE = no modificable por tenant
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (tenant_id, name)
);

-- Roles del sistema (is_system = TRUE)
INSERT INTO roles (name, description, is_system) VALUES
('superadmin',          'Administrador de plataforma — acceso total',        TRUE),
('tenant_admin',        'Administrador del tenant — gestión completa',        TRUE),
('privacy_officer',     'Oficial de Privacidad / DPO',                       TRUE),
('auditor',             'Auditor — solo lectura',                             TRUE),
('employee_it',         'Empleado área TI — tareas técnicas',                TRUE),
('employee_general',    'Empleado general — tareas propias',                  TRUE);

CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource    VARCHAR(64) NOT NULL,   -- "assessment", "remediation", "rnbd", etc.
    action      VARCHAR(32) NOT NULL,   -- "create", "read", "update", "delete", "execute"
    description TEXT,
    
    UNIQUE (resource, action)
);

CREATE TABLE role_permissions (
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id   UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_by  UUID REFERENCES users(id),
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ,  -- Roles temporales (ej.: auditor externo)
    
    PRIMARY KEY (user_id, role_id)
);

-- ============================================================
-- SESIONES
-- ============================================================
CREATE TABLE user_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    session_token   BYTEA NOT NULL UNIQUE,  -- Token hasheado (SHA-256)
    refresh_token   BYTEA UNIQUE,
    ip_address      INET NOT NULL,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ,
    revoked_reason  VARCHAR(64)  -- "logout", "security", "admin", "expired"
);

CREATE INDEX idx_sessions_user ON user_sessions(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_token ON user_sessions(session_token) WHERE revoked_at IS NULL;
```

#### 3.1.2 Diagnósticos de Cumplimiento

```sql
-- ============================================================
-- DIAGNÓSTICOS (ASSESSMENTS)
-- ============================================================
CREATE TABLE compliance_assessments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    conducted_by        UUID NOT NULL REFERENCES users(id),
    assessment_type     VARCHAR(32) NOT NULL CHECK (assessment_type IN (
                            'full',           -- Diagnóstico completo
                            'quick',          -- Diagnóstico rápido (5 dimensiones)
                            'targeted',       -- Dimensión específica
                            'annual_review',  -- Revisión anual programada
                            'incident'        -- Post-incidente
                        )),
    status              VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN (
                            'draft', 'in_progress', 'completed', 'archived'
                        )),
    framework_version   VARCHAR(16) NOT NULL DEFAULT '1.0',
    
    -- Scores por dimensión (0-100)
    score_governance        NUMERIC(5,2) CHECK (score_governance BETWEEN 0 AND 100),
    score_legal_bases       NUMERIC(5,2) CHECK (score_legal_bases BETWEEN 0 AND 100),
    score_security          NUMERIC(5,2) CHECK (score_security BETWEEN 0 AND 100),
    score_rights_mgmt       NUMERIC(5,2) CHECK (score_rights_mgmt BETWEEN 0 AND 100),
    score_third_parties     NUMERIC(5,2) CHECK (score_third_parties BETWEEN 0 AND 100),
    score_global            NUMERIC(5,2) GENERATED ALWAYS AS (
                                score_governance    * 0.20 +
                                score_legal_bases   * 0.25 +
                                score_security      * 0.25 +
                                score_rights_mgmt   * 0.15 +
                                score_third_parties * 0.15
                            ) STORED,
    
    compliance_level    VARCHAR(16) GENERATED ALWAYS AS (
                            CASE
                                WHEN (score_governance * 0.20 + score_legal_bases * 0.25 +
                                      score_security * 0.25 + score_rights_mgmt * 0.15 +
                                      score_third_parties * 0.15) >= 85 THEN 'compliant'
                                WHEN (score_governance * 0.20 + score_legal_bases * 0.25 +
                                      score_security * 0.25 + score_rights_mgmt * 0.15 +
                                      score_third_parties * 0.15) >= 70 THEN 'partial'
                                WHEN (score_governance * 0.20 + score_legal_bases * 0.25 +
                                      score_security * 0.25 + score_rights_mgmt * 0.15 +
                                      score_third_parties * 0.15) >= 50 THEN 'non_compliant'
                                WHEN (score_governance * 0.20 + score_legal_bases * 0.25 +
                                      score_security * 0.25 + score_rights_mgmt * 0.15 +
                                      score_third_parties * 0.15) >= 30 THEN 'high_risk'
                                ELSE 'critical'
                            END
                        ) STORED,
    
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assessments_tenant ON compliance_assessments(tenant_id, created_at DESC);
CREATE INDEX idx_assessments_status ON compliance_assessments(tenant_id, status);

-- ============================================================
-- ITEMS DE DIAGNÓSTICO (preguntas y respuestas)
-- ============================================================
CREATE TABLE assessment_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id       UUID NOT NULL REFERENCES compliance_assessments(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants(id),  -- Para RLS
    dimension           VARCHAR(16) NOT NULL CHECK (dimension IN (
                            'governance', 'legal_bases', 'security',
                            'rights_mgmt', 'third_parties', 'ai_audit'
                        )),
    indicator_code      VARCHAR(32) NOT NULL,  -- 'A1', 'B3', 'C5', etc.
    question_text       TEXT NOT NULL,
    answer_value        TEXT,
    answer_score        NUMERIC(5,2) CHECK (answer_score BETWEEN 0 AND 100),
    max_score           NUMERIC(5,2) NOT NULL,
    evidence_required   BOOLEAN NOT NULL DEFAULT FALSE,
    notes               TEXT,
    rag_analysis        JSONB,  -- Análisis del motor RAG para este ítem
    rag_citations       JSONB,  -- Citas normativas del RAG
    answered_at         TIMESTAMPTZ,
    
    UNIQUE (assessment_id, indicator_code)
);

-- ============================================================
-- ARCHIVOS DE EVIDENCIA
-- ============================================================
CREATE TABLE evidence_files (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    assessment_item_id  UUID REFERENCES assessment_items(id),
    remediation_task_id UUID REFERENCES remediation_tasks(id),
    uploaded_by         UUID NOT NULL REFERENCES users(id),
    
    filename_original   VARCHAR(512) NOT NULL,
    filename_stored     VARCHAR(512) NOT NULL UNIQUE,  -- UUID-based, sin info de tenant
    file_size_bytes     BIGINT NOT NULL,
    mime_type           VARCHAR(128) NOT NULL,
    storage_bucket      VARCHAR(128) NOT NULL DEFAULT 'evidence-files',
    storage_path        TEXT NOT NULL,  -- Path en MinIO/S3
    
    -- Integridad del archivo
    sha256_hash         CHAR(64) NOT NULL,  -- Para verificar integridad en descarga
    
    -- Metadatos de clasificación
    evidence_type       VARCHAR(32) CHECK (evidence_type IN (
                            'policy_document', 'dpa_contract', 'training_record',
                            'dpia_report', 'audit_report', 'screenshot',
                            'certificate', 'correspondence', 'other'
                        )),
    description         TEXT,
    valid_from          DATE,
    valid_until         DATE,
    
    -- Estado de virus scan
    virus_scan_status   VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (virus_scan_status IN (
                            'pending', 'clean', 'infected', 'error'
                        )),
    virus_scan_at       TIMESTAMPTZ,
    
    uploaded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_evidence_tenant ON evidence_files(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_assessment ON evidence_files(assessment_item_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_task ON evidence_files(remediation_task_id) WHERE deleted_at IS NULL;
```

#### 3.1.3 Log de Auditoría Inmutable

```sql
-- ============================================================
-- AUDIT LOG — INMUTABLE, APPEND-ONLY
-- NUNCA se permite UPDATE ni DELETE en esta tabla
-- ============================================================
CREATE TABLE audit_log (
    id              BIGSERIAL PRIMARY KEY,  -- SERIAL, no UUID, para ordering absoluto
    event_id        UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Actor
    actor_id        UUID,       -- NULL si es sistema/automatización
    actor_email     VARCHAR(320),
    actor_ip        INET,
    actor_user_agent TEXT,
    tenant_id       UUID,       -- NULL si es evento de plataforma
    
    -- Evento
    event_type      VARCHAR(64) NOT NULL,  -- 'user.login', 'assessment.created', etc.
    event_category  VARCHAR(32) NOT NULL CHECK (event_category IN (
                        'auth', 'assessment', 'remediation', 'data_access',
                        'admin', 'ai_query', 'document', 'security'
                    )),
    severity        VARCHAR(8) NOT NULL DEFAULT 'info' CHECK (severity IN (
                        'debug', 'info', 'warn', 'error', 'critical'
                    )),
    
    -- Objeto afectado
    resource_type   VARCHAR(64),   -- 'assessment', 'user', 'document', etc.
    resource_id     UUID,
    
    -- Detalle del evento
    action          VARCHAR(32),   -- 'create', 'read', 'update', 'delete', 'execute'
    old_values      JSONB,         -- Estado anterior (para cambios)
    new_values      JSONB,         -- Estado nuevo (para cambios)
    metadata        JSONB,         -- Metadata adicional del evento
    
    -- Integridad de la cadena de log
    previous_hash   CHAR(64),      -- Hash del evento anterior (cadena de bloques simple)
    event_hash      CHAR(64) NOT NULL,  -- SHA-256 de este evento + previous_hash
    
    CONSTRAINT audit_log_no_update CHECK (TRUE)
    -- La inmutabilidad se garantiza via REVOKE UPDATE, DELETE ON audit_log
    -- Y via Row Level Security
);

-- REVOCACIÓN DE PERMISOS DE MODIFICACIÓN (ejecutar como superuser)
REVOKE UPDATE, DELETE, TRUNCATE ON audit_log FROM PUBLIC;
REVOKE UPDATE, DELETE, TRUNCATE ON audit_log FROM app_user;  -- Usuario de la app

-- Solo se permite INSERT
GRANT INSERT, SELECT ON audit_log TO app_user;

-- Índices para consultas frecuentes de auditoría
CREATE INDEX idx_audit_tenant_time ON audit_log(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_actor ON audit_log(actor_id, timestamp DESC);
CREATE INDEX idx_audit_event_type ON audit_log(event_type, timestamp DESC);
CREATE INDEX idx_audit_resource ON audit_log(resource_type, resource_id, timestamp DESC);
```

#### 3.1.4 Documentos Legales y Embeddings

```sql
-- ============================================================
-- DOCUMENTOS LEGALES (CORPUS)
-- ============================================================
CREATE TABLE legal_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_law      VARCHAR(32) NOT NULL,
    title           TEXT NOT NULL,
    jurisdiction    CHAR(2) NOT NULL,
    language        CHAR(2) NOT NULL DEFAULT 'es',
    effective_date  DATE NOT NULL,
    expiry_date     DATE,
    version         VARCHAR(32) NOT NULL DEFAULT '1.0',
    is_current      BOOLEAN NOT NULL DEFAULT TRUE,
    source_url      TEXT,
    ingested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ingested_by     UUID REFERENCES users(id),
    file_path       TEXT  -- Path en MinIO/S3 del documento original
);

CREATE INDEX idx_legal_docs_source ON legal_documents(source_law, is_current);

-- ============================================================
-- CHUNKS DE DOCUMENTOS
-- ============================================================
CREATE TABLE document_chunks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id         UUID NOT NULL REFERENCES legal_documents(id) ON DELETE CASCADE,
    tenant_id           UUID REFERENCES tenants(id),  -- NULL = corpus público
    
    content             TEXT NOT NULL,
    content_hash        CHAR(64) NOT NULL UNIQUE,
    content_tsv         TSVECTOR GENERATED ALWAYS AS (
                            to_tsvector('spanish', content)
                        ) STORED,
    
    -- Metadata normativa
    source_law          VARCHAR(32) NOT NULL,
    article_number      VARCHAR(32),
    paragraph_number    VARCHAR(32),
    section_title       TEXT,
    jurisdiction        CHAR(2) NOT NULL,
    effective_date      DATE NOT NULL,
    effective_until     DATE,  -- NULL = sin fecha de expiración
    version             VARCHAR(32) NOT NULL DEFAULT '1.0',
    
    -- Seguridad multitenant
    sensitivity_level   INTEGER NOT NULL DEFAULT 1 CHECK (sensitivity_level BETWEEN 1 AND 5),
    
    -- Calidad y metadata técnica
    token_count         INTEGER NOT NULL,
    embedding_model     VARCHAR(64) NOT NULL,
    language            CHAR(2) NOT NULL DEFAULT 'es',
    quality_score       NUMERIC(3,2) CHECK (quality_score BETWEEN 0 AND 1),
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_document ON document_chunks(document_id);
CREATE INDEX idx_chunks_tenant ON document_chunks(tenant_id);
CREATE INDEX idx_chunks_source_law ON document_chunks(source_law, jurisdiction);
CREATE INDEX idx_chunks_tsvector ON document_chunks USING GIN(content_tsv);

-- ============================================================
-- EMBEDDINGS VECTORIALES
-- ============================================================
CREATE TABLE document_embeddings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id        UUID NOT NULL REFERENCES document_chunks(id) ON DELETE CASCADE,
    tenant_id       UUID REFERENCES tenants(id),  -- Reflejo del chunk para RLS
    embedding       vector(1536) NOT NULL,         -- Dimensión para ada-002/3-small
    model_version   VARCHAR(64) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE (chunk_id, model_version)
);

-- Índice HNSW para corpus público (estático, alta demanda de velocidad)
CREATE INDEX CONCURRENTLY idx_embeddings_hnsw_public
ON document_embeddings
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64)
WHERE tenant_id IS NULL;

-- Índice IVFFlat para documentos de tenants (actualizaciones frecuentes)
CREATE INDEX CONCURRENTLY idx_embeddings_ivfflat_tenant
ON document_embeddings
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100)
WHERE tenant_id IS NOT NULL;

-- ============================================================
-- TAREAS DE REMEDIACIÓN
-- ============================================================
CREATE TABLE remediation_tasks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    assessment_id       UUID REFERENCES compliance_assessments(id),
    assessment_item_id  UUID REFERENCES assessment_items(id),
    
    title               VARCHAR(256) NOT NULL,
    description         TEXT NOT NULL,
    priority            VARCHAR(8) NOT NULL DEFAULT 'medium' CHECK (priority IN (
                            'critical', 'high', 'medium', 'low'
                        )),
    status              VARCHAR(16) NOT NULL DEFAULT 'backlog' CHECK (status IN (
                            'backlog', 'in_progress', 'review', 'completed', 'cancelled'
                        )),
    
    -- Responsabilidad
    owner_id            UUID REFERENCES users(id),
    reviewer_id         UUID REFERENCES users(id),
    
    -- Plazos
    due_date            DATE,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    
    -- Contexto legal
    legal_reference     TEXT,      -- "Art. 5 Ley 1581 — Principio de Legalidad"
    risk_level          VARCHAR(8) CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    
    -- Metadata
    created_by          UUID NOT NULL REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_tenant ON remediation_tasks(tenant_id, status);
CREATE INDEX idx_tasks_owner ON remediation_tasks(owner_id, status);
CREATE INDEX idx_tasks_due ON remediation_tasks(tenant_id, due_date) 
    WHERE status NOT IN ('completed', 'cancelled');

-- ============================================================
-- NOTIFICACIONES
-- ============================================================
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    
    type            VARCHAR(32) NOT NULL,  -- 'task_assigned', 'rnbd_expiring', etc.
    title           VARCHAR(256) NOT NULL,
    body            TEXT NOT NULL,
    action_url      TEXT,
    
    is_read         BOOLEAN NOT NULL DEFAULT FALSE,
    read_at         TIMESTAMPTZ,
    
    -- Canales de entrega
    email_sent      BOOLEAN NOT NULL DEFAULT FALSE,
    email_sent_at   TIMESTAMPTZ,
    push_sent       BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ  -- Notificaciones temporales
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);
```

---

### 3.2 Políticas de Seguridad

#### 3.2.1 Row Level Security (RLS) — Aislamiento de Tenant

```sql
-- ============================================================
-- ACTIVAR RLS EN TODAS LAS TABLAS CON DATOS DE TENANT
-- ============================================================

ALTER TABLE compliance_assessments ENABLE ROW LEVEL SECURITY;
ALTER TABLE assessment_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE evidence_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE remediation_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_embeddings ENABLE ROW LEVEL SECURITY;

-- Variable de sesión que el backend establece ANTES de cualquier query
-- SET LOCAL app.current_tenant_id = '{tenant_id}';
-- SET LOCAL app.current_user_id = '{user_id}';
-- SET LOCAL app.current_role = '{role}';

-- ============================================================
-- POLÍTICAS RLS — compliance_assessments
-- ============================================================
CREATE POLICY tenant_isolation_assessments ON compliance_assessments
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Superadmin puede ver todo (sin filtro de tenant)
CREATE POLICY superadmin_all_assessments ON compliance_assessments
    FOR ALL
    TO app_superadmin_role  -- Rol de DB del superadmin
    USING (TRUE);

-- ============================================================
-- POLÍTICAS RLS — document_chunks (corpus público + tenant privado)
-- ============================================================
CREATE POLICY chunk_access_policy ON document_chunks
    FOR SELECT
    USING (
        -- Corpus público: visible para todos los usuarios autenticados
        tenant_id IS NULL
        OR
        -- Documentos del tenant: solo para ese tenant
        tenant_id = current_setting('app.current_tenant_id')::UUID
    );

-- Inserción de chunks: solo para corpus propio del tenant
CREATE POLICY chunk_insert_policy ON document_chunks
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('app.current_tenant_id')::UUID
        OR
        -- Superadmin puede insertar corpus público
        current_setting('app.current_role') = 'superadmin'
    );

-- ============================================================
-- POLÍTICAS RLS — document_embeddings
-- ============================================================
CREATE POLICY embedding_access_policy ON document_embeddings
    FOR SELECT
    USING (
        tenant_id IS NULL  -- Corpus público
        OR
        tenant_id = current_setting('app.current_tenant_id')::UUID
    );

-- ============================================================
-- FUNCIÓN AUXILIAR: Establecer contexto de sesión seguro
-- ============================================================
CREATE OR REPLACE FUNCTION set_tenant_context(
    p_tenant_id UUID,
    p_user_id UUID,
    p_role TEXT
) RETURNS VOID
LANGUAGE plpgsql
SECURITY DEFINER  -- Se ejecuta con permisos del owner de la función
AS $$
BEGIN
    -- Validar que el usuario pertenece al tenant
    IF NOT EXISTS (
        SELECT 1 FROM users 
        WHERE id = p_user_id 
        AND tenant_id = p_tenant_id 
        AND is_active = TRUE
        AND deleted_at IS NULL
    ) AND p_role != 'superadmin' THEN
        RAISE EXCEPTION 'Usuario no pertenece al tenant especificado';
    END IF;
    
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, TRUE);
    PERFORM set_config('app.current_user_id', p_user_id::TEXT, TRUE);
    PERFORM set_config('app.current_role', p_role, TRUE);
END;
$$;
```

#### 3.2.2 Cifrado AES-256 — Campos Sensibles

```sql
-- ============================================================
-- CIFRADO A NIVEL DE COLUMNA (AES-256-GCM)
-- Usando pgcrypto + función wrapper
-- ============================================================

-- La clave maestra NUNCA se almacena en la DB
-- Se gestiona via AWS KMS, HashiCorp Vault, o Google Cloud KMS
-- Ejemplo con función que recibe la clave como parámetro de sesión

CREATE OR REPLACE FUNCTION encrypt_sensitive(
    plaintext TEXT,
    encryption_key BYTEA  -- Derivada del KMS, pasada por la app
) RETURNS BYTEA AS $$
BEGIN
    -- AES-256 en modo GCM via pgcrypto
    -- IV aleatorio de 12 bytes generado automáticamente
    RETURN pgp_sym_encrypt(
        plaintext,
        encode(encryption_key, 'hex'),
        'cipher-algo=aes256'
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION decrypt_sensitive(
    ciphertext BYTEA,
    encryption_key BYTEA
) RETURNS TEXT AS $$
BEGIN
    RETURN pgp_sym_decrypt(
        ciphertext,
        encode(encryption_key, 'hex')
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- CAMPOS CIFRADOS EN LA BASE DE DATOS:
-- users.phone_encrypted        → teléfono personal
-- users.mfa_totp_secret        → secreto TOTP para 2FA
-- assessment_items.rag_analysis → análisis RAG (puede contener datos sensibles del tenant)
-- evidence_files.storage_path  → NO cifrado, solo el hash; el archivo en MinIO está cifrado

-- ============================================================
-- GESTIÓN DE CLAVES — Estrategia de Envelope Encryption
-- ============================================================
-- 1. AWS KMS genera y gestiona la Customer Master Key (CMK)
-- 2. Para cada tenant, se genera una Data Encryption Key (DEK) cifrada con la CMK
-- 3. La DEK cifrada se almacena en la tabla tenant_encryption_keys
-- 4. Para operar: la app solicita a KMS descifrar la DEK, la usa en memoria, la descarta

CREATE TABLE tenant_encryption_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    encrypted_dek   BYTEA NOT NULL,  -- DEK cifrada con CMK de KMS
    key_version     INTEGER NOT NULL DEFAULT 1,
    kms_key_id      VARCHAR(256) NOT NULL,  -- ARN de la CMK en KMS
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at      TIMESTAMPTZ  -- Última rotación de la DEK
);
```

#### 3.2.3 Autenticación — OAuth 2.0 + PKCE + MFA

```python
# Flujo de autenticación — FastAPI
from fastapi import FastAPI, HTTPException, Depends
from fastapi.security import OAuth2AuthorizationCodeBearer
import jwt
import pyotp
import secrets
import hashlib
from datetime import datetime, timedelta, timezone

# ============================================================
# OAUTH 2.0 + PKCE
# ============================================================

def generate_pkce_pair() -> tuple[str, str]:
    """Genera code_verifier y code_challenge para PKCE."""
    code_verifier = secrets.token_urlsafe(64)
    code_challenge = base64url_encode(
        hashlib.sha256(code_verifier.encode()).digest()
    )
    return code_verifier, code_challenge

def create_access_token(
    user_id: str,
    tenant_id: str,
    role: str,
    permissions: list[str],
    expires_minutes: int = 15  # Token de acceso de corta vida
) -> str:
    """Genera JWT de acceso firmado con RS256."""
    now = datetime.now(timezone.utc)
    payload = {
        "sub": user_id,
        "tenant_id": tenant_id,
        "role": role,
        "permissions": permissions,
        "iat": now,
        "exp": now + timedelta(minutes=expires_minutes),
        "jti": secrets.token_hex(16),  # JWT ID único para revocación
        "type": "access"
    }
    return jwt.encode(payload, PRIVATE_KEY, algorithm="RS256")

# ============================================================
# MFA — TOTP (Time-based One-Time Password)
# ============================================================

class TOTPManager:
    """Gestiona la autenticación de dos factores con TOTP (RFC 6238)."""
    
    ISSUER = "ProteccionDatosApp"
    DIGITS = 6
    INTERVAL = 30  # segundos
    VALID_WINDOW = 1  # ventana de ±1 intervalo (tolerancia de reloj)
    
    def generate_secret(self) -> str:
        """Genera un secreto TOTP nuevo (base32, 20 bytes)."""
        return pyotp.random_base32()
    
    def get_provisioning_uri(self, secret: str, user_email: str) -> str:
        """URI para generar QR code de registro en authenticator."""
        totp = pyotp.TOTP(secret, digits=self.DIGITS, interval=self.INTERVAL)
        return totp.provisioning_uri(name=user_email, issuer_name=self.ISSUER)
    
    def verify_totp(self, secret: str, provided_code: str) -> bool:
        """Verifica el código TOTP con ventana de tolerancia."""
        totp = pyotp.TOTP(secret, digits=self.DIGITS, interval=self.INTERVAL)
        return totp.verify(provided_code, valid_window=self.VALID_WINDOW)
    
    def verify_and_prevent_replay(
        self, 
        secret: str, 
        provided_code: str,
        redis_client,
        user_id: str
    ) -> bool:
        """
        Verifica TOTP y previene reutilización del mismo código
        (replay attack prevention).
        """
        # Verificar el código
        if not self.verify_totp(secret, provided_code):
            return False
        
        # Verificar que no fue usado antes en esta ventana
        redis_key = f"totp:used:{user_id}:{provided_code}"
        if redis_client.exists(redis_key):
            return False  # Replay attack detectado
        
        # Marcar como usado (TTL = 2 * intervalo para cubrir la ventana)
        redis_client.setex(redis_key, self.INTERVAL * 2, "used")
        return True

# ============================================================
# API SECURITY — Rate Limiting + JWT Validation
# ============================================================

# Configuración de rate limiting (usando slowapi)
RATE_LIMITS = {
    "auth.login": "5/minute",           # Máximo 5 intentos de login por minuto
    "auth.forgot_password": "3/hour",   # 3 solicitudes de reset por hora
    "api.default": "60/minute",         # 60 requests por minuto por usuario
    "api.rag_query": "20/minute",       # Queries RAG son costosas
    "api.bulk_export": "5/hour",        # Exportaciones masivas limitadas
}

# Política CORS para producción
CORS_POLICY = {
    "allow_origins": [
        "https://app.protecciondatos.co",
        "https://protecciondatos.co"
    ],
    "allow_credentials": True,
    "allow_methods": ["GET", "POST", "PUT", "PATCH", "DELETE"],
    "allow_headers": ["Authorization", "Content-Type", "X-Tenant-ID", "X-Request-ID"],
    "max_age": 3600,  # Preflight cache de 1 hora
    "expose_headers": ["X-Request-ID", "X-RateLimit-Remaining"]
}
```

---

## 4. Stack Tecnológico Web

### 4.1 Frontend — Next.js 14

#### 4.1.1 Arquitectura de la Aplicación Frontend

```
ESTRUCTURA DE DIRECTORIOS (App Router — Next.js 14)

app/
├── (auth)/                    # Grupo de rutas de autenticación (sin layout principal)
│   ├── login/
│   │   └── page.tsx           # Página de login con OAuth y Magic Link
│   ├── verify-mfa/
│   │   └── page.tsx           # Verificación TOTP
│   └── layout.tsx             # Layout mínimo para auth
│
├── app/                       # Rutas protegidas de la aplicación
│   └── [tenant-slug]/         # Rutas del tenant (validación en middleware)
│       ├── dashboard/
│       │   └── page.tsx       # Dashboard principal + GeoScore
│       ├── assessments/
│       │   ├── page.tsx       # Lista de diagnósticos
│       │   ├── new/
│       │   │   └── page.tsx   # Nuevo diagnóstico (wizard multi-paso)
│       │   └── [id]/
│       │       ├── page.tsx   # Detalle del diagnóstico
│       │       └── report/
│       │           └── page.tsx # Informe PDF
│       ├── remediation/
│       │   └── page.tsx       # Tablero Kanban de remediación
│       ├── rnbd/
│       │   └── page.tsx       # Gestión RNBD
│       ├── dpia/
│       │   └── page.tsx       # DPIAs
│       ├── settings/
│       │   ├── users/
│       │   └── page.tsx       # Configuración del tenant
│       └── layout.tsx         # Layout principal con sidebar y header
│
├── api/                       # API Routes (Next.js — solo para BFF/proxy)
│   └── auth/
│       └── [...nextauth]/
│           └── route.ts       # NextAuth.js handlers
│
├── components/
│   ├── ui/                    # Componentes shadcn/ui
│   ├── dashboard/
│   │   ├── ComplianceGauge.tsx
│   │   ├── DimensionRadarChart.tsx
│   │   ├── GeoScoreMap.tsx    # Mapa de cumplimiento geográfico (Mapbox GL)
│   │   ├── TrendLineChart.tsx
│   │   └── AlertsPanel.tsx
│   ├── assessment/
│   │   ├── AssessmentWizard.tsx
│   │   ├── QuestionCard.tsx
│   │   ├── EvidenceUploader.tsx
│   │   └── RAGAnalysisPanel.tsx  # Muestra análisis del RAG en tiempo real
│   ├── remediation/
│   │   ├── KanbanBoard.tsx
│   │   ├── TaskCard.tsx
│   │   └── TaskDetailModal.tsx
│   └── shared/
│       ├── PermissionGuard.tsx
│       ├── TenantProvider.tsx
│       └── DataMask.tsx       # Enmascaramiento de PII en UI
│
├── lib/
│   ├── api-client.ts          # Cliente HTTP con interceptores de tenant + JWT
│   ├── auth.ts                # NextAuth.js config
│   ├── rbac.ts                # Lógica de permisos client-side
│   └── validators/            # Zod schemas para formularios
│
├── middleware.ts               # Validación de tenant-slug + auth en edge
└── next.config.js             # Configuración de Next.js (CSP, headers, etc.)
```

#### 4.1.2 Middleware de Seguridad (Edge)

```typescript
// middleware.ts — Ejecuta en Cloudflare Workers / Vercel Edge
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { jwtVerify } from 'jose';

export async function middleware(request: NextRequest) {
    const pathname = request.nextUrl.pathname;
    
    // Solo proteger rutas de app
    if (!pathname.startsWith('/app/')) {
        return NextResponse.next();
    }
    
    // 1. Verificar sesión JWT
    const token = request.cookies.get('session-token')?.value;
    if (!token) {
        return NextResponse.redirect(new URL('/login', request.url));
    }
    
    try {
        const { payload } = await jwtVerify(
            token,
            await importPublicKey(PUBLIC_KEY),
            { algorithms: ['RS256'] }
        );
        
        // 2. Extraer tenant-slug de la URL y validar contra el JWT
        const tenantSlug = pathname.split('/')[2];
        if (payload.tenant_slug !== tenantSlug) {
            // Intento de acceso a tenant incorrecto
            return NextResponse.json(
                { error: 'Acceso no autorizado al tenant' },
                { status: 403 }
            );
        }
        
        // 3. Propagar headers de contexto al backend
        const requestHeaders = new Headers(request.headers);
        requestHeaders.set('X-Tenant-ID', payload.tenant_id as string);
        requestHeaders.set('X-User-ID', payload.sub as string);
        requestHeaders.set('X-User-Role', payload.role as string);
        
        return NextResponse.next({ request: { headers: requestHeaders } });
        
    } catch (error) {
        // Token inválido o expirado
        const response = NextResponse.redirect(new URL('/login', request.url));
        response.cookies.delete('session-token');
        return response;
    }
}

export const config = {
    matcher: ['/app/:path*']
};
```

#### 4.1.3 Configuración de Seguridad HTTP (Next.js)

```javascript
// next.config.js
const securityHeaders = [
    {
        key: 'Content-Security-Policy',
        value: [
            "default-src 'self'",
            "script-src 'self' 'nonce-{nonce}' https://cdnjs.cloudflare.com",
            "style-src 'self' 'unsafe-inline'",  // Necesario para Tailwind CSS
            "img-src 'self' data: https:",
            "connect-src 'self' https://api.protecciondatos.co wss://api.protecciondatos.co",
            "font-src 'self'",
            "frame-src 'none'",
            "object-src 'none'",
            "base-uri 'self'",
            "form-action 'self'",
            "upgrade-insecure-requests",
        ].join('; ')
    },
    { key: 'X-Frame-Options', value: 'DENY' },
    { key: 'X-Content-Type-Options', value: 'nosniff' },
    { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
    { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
    {
        key: 'Strict-Transport-Security',
        value: 'max-age=63072000; includeSubDomains; preload'
    }
];
```

---

### 4.2 Backend — FastAPI (Servicio RAG/AI)

#### 4.2.1 Estructura del Microservicio RAG

```python
# Estructura del microservicio Python/FastAPI

ai_service/
├── app/
│   ├── main.py                     # FastAPI app + lifespan
│   ├── api/
│   │   ├── v1/
│   │   │   ├── rag_query.py        # POST /v1/rag/query
│   │   │   ├── assessment.py       # POST /v1/assessment/analyze
│   │   │   ├── ingestion.py        # POST /v1/documents/ingest (admin)
│   │   │   └── health.py           # GET /v1/health
│   ├── core/
│   │   ├── config.py               # Pydantic Settings (env vars)
│   │   ├── security.py             # JWT validation, tenant context
│   │   └── database.py             # SQLAlchemy async pool
│   ├── services/
│   │   ├── rag/
│   │   │   ├── query_processor.py  # Procesamiento de queries
│   │   │   ├── retrieval.py        # Filtering Wall + Hybrid Search
│   │   │   ├── reranker.py         # Cross-Encoder reranking
│   │   │   └── agent_graph.py      # LangGraph agent orchestration
│   │   ├── embedding/
│   │   │   ├── router.py           # EmbeddingRouter (público/privado)
│   │   │   └── models.py           # Wrappers de modelos
│   │   └── ingestion/
│   │       ├── pipeline.py         # Orquestador del pipeline
│   │       ├── chunker.py          # LegalDocumentChunker
│   │       └── metadata.py         # Enriquecimiento de metadata
│   └── models/
│       ├── schemas.py              # Pydantic models (request/response)
│       └── db_models.py            # SQLAlchemy ORM models
├── tests/
├── Dockerfile
└── requirements.txt

# Configuración del endpoint RAG
# POST /v1/rag/query

class RAGQueryRequest(BaseModel):
    query: str = Field(..., min_length=10, max_length=2000)
    jurisdictions: list[str] = Field(default=["CO"])
    include_global: bool = True
    max_chunks: int = Field(default=5, ge=1, le=20)
    assessment_id: UUID | None = None  # Para vincular al diagnóstico

class RAGQueryResponse(BaseModel):
    answer: str
    confidence_score: float
    requires_human_review: bool
    citations: list[Citation]
    chunks_used: list[ChunkReference]
    processing_time_ms: int
```

---

### 4.3 Infraestructura y Observabilidad

#### 4.3.1 Docker Compose — Desarrollo Local

```yaml
# docker-compose.yml
version: '3.9'

services:
  # ──────────────────────────────────────────
  # BASE DE DATOS PRINCIPAL
  # ──────────────────────────────────────────
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: proteccion_datos
      POSTGRES_USER: app_user
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app_user -d proteccion_datos"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          memory: 4G
        reservations:
          memory: 2G

  # ──────────────────────────────────────────
  # REDIS — CACHÉ, SESIONES Y RATE LIMITING
  # ──────────────────────────────────────────
  redis:
    image: redis:7.2-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD} --maxmemory 512mb --maxmemory-policy allkeys-lru
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  # ──────────────────────────────────────────
  # MINIO — ALMACENAMIENTO DE ARCHIVOS
  # ──────────────────────────────────────────
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ──────────────────────────────────────────
  # SERVICIO RAG/AI — FASTAPI
  # ──────────────────────────────────────────
  ai_service:
    build:
      context: ./ai_service
      dockerfile: Dockerfile
    environment:
      DATABASE_URL: postgresql+asyncpg://app_user:${POSTGRES_PASSWORD}@postgres:5432/proteccion_datos
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379/0
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      JWT_PUBLIC_KEY_PATH: /secrets/jwt_public.pem
      ENVIRONMENT: development
    volumes:
      - ./secrets:/secrets:ro
    ports:
      - "8000:8000"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    deploy:
      resources:
        limits:
          memory: 2G  # El modelo local bge-m3 requiere ~1.5GB

  # ──────────────────────────────────────────
  # GATEWAY API — NODE.JS
  # ──────────────────────────────────────────
  api_gateway:
    build:
      context: ./api_gateway
      dockerfile: Dockerfile
    environment:
      AI_SERVICE_URL: http://ai_service:8000
      DATABASE_URL: postgresql://app_user:${POSTGRES_PASSWORD}@postgres:5432/proteccion_datos
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379/1
      JWT_PRIVATE_KEY_PATH: /secrets/jwt_private.pem
      FRONTEND_URL: http://localhost:3000
    ports:
      - "4000:4000"
    depends_on:
      - postgres
      - redis
      - ai_service

  # ──────────────────────────────────────────
  # FRONTEND — NEXT.JS
  # ──────────────────────────────────────────
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.dev
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:4000
      NEXTAUTH_SECRET: ${NEXTAUTH_SECRET}
      NEXTAUTH_URL: http://localhost:3000
    volumes:
      - ./frontend:/app
      - /app/node_modules  # Excluir node_modules del volumen
    ports:
      - "3000:3000"

  # ──────────────────────────────────────────
  # OBSERVABILIDAD
  # ──────────────────────────────────────────
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./observability/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    environment:
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD}
      GF_AUTH_ANONYMOUS_ENABLED: "false"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./observability/grafana/dashboards:/etc/grafana/provisioning/dashboards
    ports:
      - "3001:3000"
    depends_on:
      - prometheus

volumes:
  postgres_data:
  redis_data:
  minio_data:
  prometheus_data:
  grafana_data:
```

#### 4.3.2 Observabilidad — OpenTelemetry

```python
# Instrumentación OpenTelemetry para el servicio RAG/AI
from opentelemetry import trace, metrics
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.sqlalchemy import SQLAlchemyInstrumentor

def setup_telemetry(app):
    """Configura OpenTelemetry para el microservicio."""
    
    # Tracer
    tracer_provider = TracerProvider(
        resource=Resource(attributes={
            SERVICE_NAME: "proteccion-datos-ai-service",
            SERVICE_VERSION: "1.0.0",
            DEPLOYMENT_ENVIRONMENT: settings.ENVIRONMENT
        })
    )
    
    # Exportar spans a Jaeger/Tempo via OTLP
    otlp_exporter = OTLPSpanExporter(
        endpoint=settings.OTEL_EXPORTER_OTLP_ENDPOINT
    )
    tracer_provider.add_span_processor(BatchSpanProcessor(otlp_exporter))
    trace.set_tracer_provider(tracer_provider)
    
    # Auto-instrumentación
    FastAPIInstrumentor.instrument_app(app)
    SQLAlchemyInstrumentor().instrument(engine=async_engine)
    
    return tracer_provider

# Métricas personalizadas para el sistema RAG
meter = metrics.get_meter("rag_service")

# Histograma de latencia de queries RAG
rag_query_latency = meter.create_histogram(
    name="rag.query.duration_ms",
    description="Latencia de queries RAG en milisegundos",
    unit="ms"
)

# Contador de queries por tenant
rag_queries_total = meter.create_counter(
    name="rag.queries.total",
    description="Total de queries RAG procesadas"
)

# Gauge de confianza promedio
rag_confidence_avg = meter.create_gauge(
    name="rag.confidence.average",
    description="Confianza promedio de respuestas RAG"
)

# Contador de escalaciones a revisión humana
rag_human_escalations = meter.create_counter(
    name="rag.human_escalations.total",
    description="Queries escaladas a revisión humana por baja confianza"
)
```

---

### 4.4 CI/CD — GitHub Actions

```yaml
# .github/workflows/ci.yml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  # ──────────────────────────────────────────
  # 1. ANÁLISIS DE SEGURIDAD
  # ──────────────────────────────────────────
  security-scan:
    name: Security Analysis
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          severity: 'CRITICAL,HIGH'
          
      - name: Run Semgrep SAST
        uses: semgrep/semgrep-action@v1
        with:
          config: >-
            p/security-audit
            p/owasp-top-ten
            p/python
            p/typescript
          
      - name: Check for secrets
        uses: trufflesecurity/trufflehog@main
        with:
          path: ./
          base: ${{ github.event.repository.default_branch }}

  # ──────────────────────────────────────────
  # 2. TESTS — FRONTEND
  # ──────────────────────────────────────────
  frontend-tests:
    name: Frontend Tests
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: ./frontend
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json
          
      - run: npm ci
      - run: npm run type-check
      - run: npm run lint
      - run: npm run test:unit -- --coverage
      - run: npm run test:e2e  # Playwright

  # ──────────────────────────────────────────
  # 3. TESTS — AI SERVICE
  # ──────────────────────────────────────────
  ai-service-tests:
    name: AI Service Tests
    runs-on: ubuntu-latest
    services:
      postgres:
        image: pgvector/pgvector:pg16
        env:
          POSTGRES_DB: test_db
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_password
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    defaults:
      run:
        working-directory: ./ai_service
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
          cache: 'pip'
          
      - run: pip install -r requirements.txt
      - run: pip install -r requirements-test.txt
      - run: ruff check .       # Linting
      - run: mypy app/          # Type checking
      - run: pytest tests/ -v --cov=app --cov-report=xml
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4

  # ──────────────────────────────────────────
  # 4. BUILD Y PUSH DE IMÁGENES DOCKER
  # ──────────────────────────────────────────
  build-and-push:
    name: Build Docker Images
    needs: [security-scan, frontend-tests, ai-service-tests]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_DEPLOY_ROLE_ARN }}
          aws-region: us-east-1
          
      - name: Login to Amazon ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v2
        
      - name: Build and push AI Service image
        env:
          ECR_REGISTRY: ${{ steps.login-ecr.outputs.registry }}
          IMAGE_TAG: ${{ github.sha }}
        run: |
          docker build -t $ECR_REGISTRY/ai-service:$IMAGE_TAG ./ai_service
          docker push $ECR_REGISTRY/ai-service:$IMAGE_TAG
          docker tag $ECR_REGISTRY/ai-service:$IMAGE_TAG $ECR_REGISTRY/ai-service:latest
          docker push $ECR_REGISTRY/ai-service:latest

  # ──────────────────────────────────────────
  # 5. DEPLOY A KUBERNETES (EKS)
  # ──────────────────────────────────────────
  deploy:
    name: Deploy to EKS
    needs: [build-and-push]
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    environment: production
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Configure kubectl
        uses: aws-actions/amazon-eks-update-kubeconfig@v1
        with:
          cluster-name: proteccion-datos-cluster
          region: us-east-1
          
      - name: Deploy with Helm
        run: |
          helm upgrade --install proteccion-datos ./helm/proteccion-datos \
            --namespace production \
            --set aiService.image.tag=${{ github.sha }} \
            --set frontend.image.tag=${{ github.sha }} \
            --set apiGateway.image.tag=${{ github.sha }} \
            --wait \
            --timeout 10m
```

---

### 4.5 Kubernetes — Recursos de Producción

```yaml
# helm/proteccion-datos/templates/ai-service-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-service
  namespace: production
  labels:
    app: ai-service
    version: "1.0.0"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ai-service
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0  # Zero-downtime deployments
  template:
    metadata:
      labels:
        app: ai-service
    spec:
      serviceAccountName: ai-service-sa
      
      # Seguridad del Pod
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 2000
        seccompProfile:
          type: RuntimeDefault
      
      containers:
        - name: ai-service
          image: "{{ .Values.ecrRegistry }}/ai-service:{{ .Values.aiService.image.tag }}"
          
          ports:
            - containerPort: 8000
          
          # Seguridad del contenedor
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop: ["ALL"]
          
          # Variables de entorno desde Secrets
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: ai-service-secrets
                  key: database-url
            - name: OPENAI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: ai-service-secrets
                  key: openai-api-key
          
          # Recursos (importante para el modelo bge-m3)
          resources:
            requests:
              cpu: "500m"
              memory: "2Gi"
            limits:
              cpu: "2000m"
              memory: "4Gi"
          
          # Health checks
          livenessProbe:
            httpGet:
              path: /v1/health/live
              port: 8000
            initialDelaySeconds: 30
            periodSeconds: 10
            failureThreshold: 3
          
          readinessProbe:
            httpGet:
              path: /v1/health/ready
              port: 8000
            initialDelaySeconds: 15
            periodSeconds: 5
          
          volumeMounts:
            - name: secrets-volume
              mountPath: /secrets
              readOnly: true
            - name: tmp-volume
              mountPath: /tmp
      
      volumes:
        - name: secrets-volume
          secret:
            secretName: ai-service-tls-secrets
        - name: tmp-volume
          emptyDir: {}

---
# Horizontal Pod Autoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ai-service-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ai-service
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80

---
# PodDisruptionBudget — garantizar disponibilidad durante mantenimiento
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: ai-service-pdb
  namespace: production
spec:
  minAvailable: 2  # Siempre mínimo 2 pods disponibles
  selector:
    matchLabels:
      app: ai-service
```

---

### 4.6 Estrategia de Multi-Región para Residencia de Datos

```yaml
# Configuración de regiones por jurisdicción
DATA_RESIDENCY_CONFIG:
  CO:  # Colombia
    primary_region: sa-east-1      # São Paulo (más cercano disponible)
    backup_region: us-east-1       # Virginia (fallback)
    compliance_notes:
      - "Ley 1581 no exige expresamente residencia local, pero SIC recomienda"
      - "Datos de entidades públicas: preferiblemente en Colombia o región andina"
    rds_config:
      instance_type: db.r6g.xlarge
      multi_az: true
      storage_encrypted: true
      kms_key_id: "arn:aws:kms:sa-east-1:ACCOUNT:key/KEY_ID"
  
  EU:  # Clientes con datos de ciudadanos UE
    primary_region: eu-west-1      # Irlanda
    backup_region: eu-central-1    # Frankfurt
    compliance_notes:
      - "GDPR Art. 44-49: datos deben permanecer en UE o país con adecuación"
      - "Schrems II: TIA obligatorio para cualquier transferencia fuera de EEE"
    rds_config:
      instance_type: db.r6g.xlarge
      multi_az: true
      storage_encrypted: true

# Política de enrutamiento — CloudFront + Route53
routing_policy:
  tenant_onboarding:
    - step: "Al crear tenant, registrar data_residency_region"
    - step: "Crear namespace Kubernetes en la región correspondiente"
    - step: "Provisionar instancia RDS dedicada o usar pool de la región"
    - step: "Configurar MinIO bucket en la región correcta"
    - step: "Registrar en Route53 el subdomain del tenant apuntando a la región"
  
  cross_region_restrictions:
    - "Datos de tenant CO NUNCA se replican a EU sin consentimiento explícito"
    - "Datos de tenant EU NUNCA se replican a CO (Schrems II)"
    - "El corpus legal público (Ley 1581, GDPR) puede estar en múltiples regiones"
    - "Los embeddings del tenant son tan sensibles como los datos — misma región"
```

---

*Fin del documento TDD_Arquitectura_Tecnica_y_RAG.md*  
*Versión 1.0.0 — 2026-06-26*
