-- Datos de desarrollo: tenant demo + corpus Ley 1581 (fragmentos)
INSERT INTO tenants (id, name, slug, legal_name, company_size, legal_nature)
VALUES (
    'a0000000-0000-4000-8000-000000000001',
    'Empresa Demo SAS',
    'empresa-demo',
    'Empresa Demo SAS',
    'medium',
    'private_medium'
);

INSERT INTO users (id, tenant_id, email, display_name)
VALUES (
    'b0000000-0000-4000-8000-000000000001',
    'a0000000-0000-4000-8000-000000000001',
    'dpo@empresa-demo.co',
    'Oficial de Privacidad Demo'
);

INSERT INTO user_roles (user_id, role_id)
SELECT 'b0000000-0000-4000-8000-000000000001', id
FROM roles WHERE name = 'privacy_officer' AND is_system = TRUE;

INSERT INTO legal_documents (id, source_law, title, jurisdiction, effective_date)
VALUES (
    'c0000000-0000-4000-8000-000000000001',
    'LEY_1581_2012',
    'Ley 1581 de 2012 — Protección de Datos Personales',
    'CO',
    '2012-10-18'
);

INSERT INTO document_chunks (
    document_id, tenant_id, content, content_hash, source_law,
    article_number, jurisdiction, effective_date, token_count
) VALUES
(
    'c0000000-0000-4000-8000-000000000001',
    NULL,
    'Artículo 4. Principios rectores. El tratamiento de datos personales debe cumplir los siguientes principios: a) Legalidad; b) Finalidad; c) Libertad; d) Veracidad o calidad; e) Transparencia; f) Acceso y circulación restringida; g) Seguridad; h) Confidencialidad.',
    encode(digest('ley1581-art4', 'sha256'), 'hex'),
    'LEY_1581_2012',
    '4',
    'CO',
    '2012-10-18',
    45
),
(
    'c0000000-0000-4000-8000-000000000001',
    NULL,
    'Artículo 9. Autorización. Requiere autorización previa, expresa e informada del Titular para recolectar, almacenar, usar, circular o suprimir sus datos personales, salvo las excepciones previstas en la ley.',
    encode(digest('ley1581-art9', 'sha256'), 'hex'),
    'LEY_1581_2012',
    '9',
    'CO',
    '2012-10-18',
    38
),
(
    'c0000000-0000-4000-8000-000000000001',
    NULL,
    'Artículo 15. Información al Titular. El responsable del tratamiento debe informar al Titular sobre la existencia de la política de tratamiento y los cambios sustanciales, indicando finalidades, derechos y mecanismos para ejercerlos.',
    encode(digest('ley1581-art15', 'sha256'), 'hex'),
    'LEY_1581_2012',
    '15',
    'CO',
    '2012-10-18',
    42
);

INSERT INTO compliance_assessments (
    id, tenant_id, conducted_by, assessment_type, status,
    score_governance, score_legal_bases, score_security,
    score_rights_mgmt, score_third_parties, started_at
) VALUES (
    'd0000000-0000-4000-8000-000000000001',
    'a0000000-0000-4000-8000-000000000001',
    'b0000000-0000-4000-8000-000000000001',
    'quick',
    'in_progress',
    72, 65, 80, 58, 70,
    NOW()
);

INSERT INTO remediation_tasks (
    tenant_id, assessment_id, title, description, priority, status,
    legal_reference, created_by, due_date
) VALUES (
    'a0000000-0000-4000-8000-000000000001',
    'd0000000-0000-4000-8000-000000000001',
    'Actualizar Política de Privacidad',
    'Completar sección de finalidades y derechos ARCO según Art. 15 Ley 1581.',
    'high',
    'in_progress',
    'Art. 15 Ley 1581/2012',
    'b0000000-0000-4000-8000-000000000001',
    CURRENT_DATE + INTERVAL '30 days'
);
