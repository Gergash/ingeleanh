-- Tenants y usuarios
CREATE TABLE tenants (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    VARCHAR(256) NOT NULL,
    slug                    VARCHAR(128) NOT NULL UNIQUE,
    nit                     VARCHAR(20),
    legal_name              VARCHAR(512) NOT NULL,
    country                 CHAR(2) NOT NULL DEFAULT 'CO',
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
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    email               VARCHAR(320) NOT NULL,
    display_name        VARCHAR(256),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, email)
);

CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),
    name        VARCHAR(64) NOT NULL,
    description TEXT,
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (tenant_id, name)
);

INSERT INTO roles (name, description, is_system) VALUES
('superadmin', 'Administrador de plataforma', TRUE),
('tenant_admin', 'Administrador del tenant', TRUE),
('privacy_officer', 'Oficial de Privacidad / DPO', TRUE),
('auditor', 'Auditor — solo lectura', TRUE);

CREATE TABLE user_roles (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- Diagnósticos
CREATE TABLE compliance_assessments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    conducted_by        UUID NOT NULL REFERENCES users(id),
    assessment_type     VARCHAR(32) NOT NULL DEFAULT 'quick',
    status              VARCHAR(16) NOT NULL DEFAULT 'draft',
    score_governance        NUMERIC(5,2) CHECK (score_governance BETWEEN 0 AND 100),
    score_legal_bases       NUMERIC(5,2) CHECK (score_legal_bases BETWEEN 0 AND 100),
    score_security          NUMERIC(5,2) CHECK (score_security BETWEEN 0 AND 100),
    score_rights_mgmt       NUMERIC(5,2) CHECK (score_rights_mgmt BETWEEN 0 AND 100),
    score_third_parties     NUMERIC(5,2) CHECK (score_third_parties BETWEEN 0 AND 100),
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE assessment_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id       UUID NOT NULL REFERENCES compliance_assessments(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    dimension           VARCHAR(16) NOT NULL,
    indicator_code      VARCHAR(32) NOT NULL,
    question_text       TEXT NOT NULL,
    answer_value        TEXT,
    answer_score        NUMERIC(5,2),
    max_score           NUMERIC(5,2) NOT NULL,
    rag_analysis        JSONB,
    rag_citations       JSONB,
    UNIQUE (assessment_id, indicator_code)
);

CREATE TABLE remediation_tasks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    assessment_id       UUID REFERENCES compliance_assessments(id),
    title               VARCHAR(256) NOT NULL,
    description         TEXT NOT NULL,
    priority            VARCHAR(8) NOT NULL DEFAULT 'medium',
    status              VARCHAR(16) NOT NULL DEFAULT 'backlog',
    owner_id            UUID REFERENCES users(id),
    due_date            DATE,
    legal_reference     TEXT,
    created_by          UUID NOT NULL REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Corpus legal y RAG
CREATE TABLE legal_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_law      VARCHAR(32) NOT NULL,
    title           TEXT NOT NULL,
    jurisdiction    CHAR(2) NOT NULL,
    language        CHAR(2) NOT NULL DEFAULT 'es',
    effective_date  DATE NOT NULL,
    version         VARCHAR(32) NOT NULL DEFAULT '1.0',
    is_current      BOOLEAN NOT NULL DEFAULT TRUE,
    ingested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_chunks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id         UUID NOT NULL REFERENCES legal_documents(id) ON DELETE CASCADE,
    tenant_id           UUID REFERENCES tenants(id),
    content             TEXT NOT NULL,
    content_hash        CHAR(64) NOT NULL UNIQUE,
    content_tsv         TSVECTOR GENERATED ALWAYS AS (to_tsvector('spanish', content)) STORED,
    source_law          VARCHAR(32) NOT NULL,
    article_number      VARCHAR(32),
    jurisdiction        CHAR(2) NOT NULL,
    effective_date      DATE NOT NULL,
    sensitivity_level   INTEGER NOT NULL DEFAULT 1 CHECK (sensitivity_level BETWEEN 1 AND 5),
    token_count         INTEGER NOT NULL DEFAULT 0,
    embedding_model     VARCHAR(64) NOT NULL DEFAULT 'text-embedding-3-small',
    language            CHAR(2) NOT NULL DEFAULT 'es',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_tsvector ON document_chunks USING GIN(content_tsv);
CREATE INDEX idx_chunks_tenant ON document_chunks(tenant_id);
CREATE INDEX idx_assessments_tenant ON compliance_assessments(tenant_id, created_at DESC);

CREATE TABLE document_embeddings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chunk_id        UUID NOT NULL REFERENCES document_chunks(id) ON DELETE CASCADE,
    tenant_id       UUID REFERENCES tenants(id),
    embedding       vector(1536),
    model_version   VARCHAR(64) NOT NULL DEFAULT 'text-embedding-3-small',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chunk_id, model_version)
);
