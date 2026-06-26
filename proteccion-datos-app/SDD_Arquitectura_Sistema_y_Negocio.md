# SDD — Arquitectura de Sistema y Negocio
## Plataforma SaaS de Autodiagnóstico de Protección de Datos

**Versión:** 1.0.0  
**Fecha:** 2026-06-26  
**Clasificación:** Confidencial — Uso Interno  
**Marco normativo:** Ley 1581/2012 (Colombia) · GDPR (UE) 2016/679 · ISO/IEC 27701:2019  

---

## Tabla de Contenido

1. [Ontología y Modelo de Dominio Legal](#1-ontología-y-modelo-de-dominio-legal)
2. [Motor de Responsabilidad Demostrada (Accountability)](#2-motor-de-responsabilidad-demostrada-accountability)
3. [Gobernanza de Flujos Transfronterizos y Tecnológicos](#3-gobernanza-de-flujos-transfronterizos-y-tecnológicos)
4. [Auditoría de IA y Algoritmos](#4-auditoría-de-ia-y-algoritmos)
5. [Arquitectura de UI/UX y Multitenencia](#5-arquitectura-de-uiux-y-multitenencia)

---

## 1. Ontología y Modelo de Dominio Legal

### 1.1 Principios Rectores y su Mapeo al Sistema

La plataforma implementa un motor de evaluación basado en grafos de principios que traduce las obligaciones normativas abstractas en dimensiones auditables concretas. Los principios son evaluados de manera interdependiente, reconociendo que la violación de uno generalmente implica la de otros.

#### 1.1.1 Principio de Legalidad

| Dimensión | Fuente Normativa | Elemento Evaluable | Peso en Score |
|---|---|---|---|
| Base jurídica del tratamiento | Ley 1581 Art. 4(a), GDPR Art. 6 | Existencia y validez de la base legal documentada | 15% |
| Registro ante autoridad | Decreto 1377/2013, SIC | Inscripción vigente en RNBD | 10% |
| Políticas de Privacidad | Ley 1581 Art. 15, GDPR Art. 13/14 | Publicación, completitud y actualización | 8% |
| Autorización de tratamiento | Ley 1581 Art. 9 | Mecanismos de obtención, almacenamiento y prueba | 12% |

**Árbol de decisión — Validación de Base Jurídica (Ley 1581 + GDPR):**

```
¿Existe base jurídica documentada para cada finalidad?
├── NO → CRÍTICO (incumplimiento inmediato — Art. 4(a) Ley 1581)
└── SÍ → ¿Es la base aplicable según la naturaleza del dato?
         ├── Dato sensible + base = consentimiento
         │   └── ¿Es expreso, previo, informado y específico? (Art. 9 GDPR)
         │       ├── SÍ → CONFORME
         │       └── NO → ALTO RIESGO
         ├── Dato personal ordinario + base = contrato
         │   └── ¿El tratamiento es estrictamente necesario para la ejecución?
         │       ├── SÍ → CONFORME
         │       └── NO → REVISAR (puede requerir consentimiento)
         └── Interés legítimo (solo GDPR Art. 6(1)(f))
             └── ¿Se realizó test LIA (Legitimate Interest Assessment)?
                 ├── SÍ, documentado → CONFORME CON SEGUIMIENTO
                 └── NO → ADVERTENCIA — riesgo de impugnación
```

#### 1.1.2 Principio de Finalidad

El sistema evalúa si cada tratamiento de datos tiene una finalidad:
- **Determinada:** especificada antes o en el momento de la recolección
- **Explícita:** comunicada al titular en lenguaje claro y comprensible
- **Legítima:** compatible con el ordenamiento jurídico
- **Limitada:** no extensible a finalidades incompatibles sin nueva autorización

**Matriz de Evaluación de Finalidades:**

```json
{
  "evaluacion_finalidad": {
    "dimensiones": {
      "determinacion": {
        "pregunta": "¿Cada base de datos tiene finalidades documentadas antes de iniciar el tratamiento?",
        "evidencia_requerida": ["política_privacidad_vigente", "aviso_privacidad", "registro_rnbd"],
        "puntaje_max": 25
      },
      "explicitud": {
        "pregunta": "¿Las finalidades son comprensibles para un ciudadano promedio (nivel Flesch-Kincaid adecuado)?",
        "evidencia_requerida": ["texto_politica", "analisis_legibilidad"],
        "puntaje_max": 25
      },
      "legitimidad": {
        "pregunta": "¿Las finalidades no contravienen la Constitución, la ley ni el orden público?",
        "evidencia_requerida": ["concepto_juridico", "revision_dpo"],
        "puntaje_max": 25
      },
      "limitacion": {
        "pregunta": "¿Existe control técnico que impide usar los datos para finalidades no declaradas?",
        "evidencia_requerida": ["controles_acceso", "logs_auditoria", "politica_uso_interno"],
        "puntaje_max": 25
      }
    },
    "umbral_conformidad": 70,
    "umbral_riesgo_alto": 40
  }
}
```

#### 1.1.3 Principio de Libertad

El tratamiento de datos personales solo puede realizarse con el consentimiento previo, expreso e informado del titular, excepto en los casos previstos en la ley. El sistema evalúa la ausencia de condicionamiento, coacción o engaño en la obtención de la autorización.

**Indicadores de vulneración del principio de libertad detectados por el sistema:**
- Dark patterns en formularios de consentimiento (pre-marcados, fuentes pequeñas, ocultamiento)
- Bundling de consentimiento (aceptar todo o nada sin granularidad)
- Consentimiento como condición para acceder a servicio que no lo requiere
- Ausencia de mecanismo de revocación accesible
- Renovación de consentimiento no implementada cuando cambian finalidades

#### 1.1.4 Principio de Veracidad / Calidad del Dato

| Control | Descripción | Frecuencia de Auditoría |
|---|---|---|
| Exactitud | ¿Los datos corresponden a la situación real del titular? | Continua (triggers automáticos) |
| Completitud | ¿Los registros tienen todos los campos necesarios para la finalidad? | Mensual |
| Actualización | ¿Existen procedimientos para corregir datos desactualizados? | Trimestral |
| Pertinencia | ¿Solo se recolectan datos necesarios para la finalidad declarada? | Por evento de nueva base |
| Retención | ¿Se eliminan los datos cuando se cumple la finalidad o vence el plazo legal? | Semestral |

#### 1.1.5 Principio de Transparencia

El sistema evalúa la transparencia en dos capas:

**Capa Externa (hacia titulares):**
- Disponibilidad y accesibilidad de la Política de Privacidad (formato, idioma, canal)
- Completitud según Art. 15 Ley 1581 y Arts. 13/14 GDPR
- Mecanismos para ejercicio de derechos ARCO (Acceso, Rectificación, Cancelación, Oposición) + portabilidad y olvido (GDPR)
- Tiempos de respuesta a solicitudes de titulares (legal: 10 días hábiles Ley 1581, 30 días GDPR)

**Capa Interna (hacia autoridades):**
- Registros de actividades de tratamiento (Art. 30 GDPR, RNBD Colombia)
- Disponibilidad de documentación para inspección
- Canal de comunicación con la SIC (Superintendencia de Industria y Comercio)

---

### 1.2 Categorización Estricta de Datos Personales

El motor de categorización es el núcleo del sistema. Implementa un clasificador de cuatro capas que combina análisis semántico (NLP) con reglas jurídicas deterministas.

#### 1.2.1 Taxonomía Legal Completa

```
DATOS PERSONALES
├── DATOS PÚBLICOS
│   ├── Definición: son de naturaleza pública, contenidos en registros
│   │   y documentos públicos (Art. 3 Ley 1581)
│   ├── Ejemplos: nombre en escritura pública, estado civil en registro civil,
│   │   oficio o profesión en tarjeta profesional, calidad de comerciante en
│   │   registro mercantil, datos de redes sociales con perfil público
│   ├── Régimen: no requieren autorización para tratamiento (Art. 10 Ley 1581)
│   ├── Restricción: la naturaleza pública del dato no autoriza tratamientos
│   │   que vulneren otros principios (finalidad, proporcionalidad)
│   └── GDPR equivalente: no aplica categoría específica; puede ser dato
│       personal ordinario bajo Art. 6(1)(f) con interés legítimo
│
├── DATOS SEMI-PRIVADOS
│   ├── Definición: no íntimos ni públicos, cuyo conocimiento o divulgación
│   │   puede interesar no solo al titular sino a cierto sector o grupo
│   ├── Ejemplos: datos laborales (cargo, empresa, salario en rangos generales),
│   │   datos académicos (institución, título), historial crediticio básico,
│   │   número de teléfono de empresa
│   ├── Régimen: requieren autorización del titular o mandato legal
│   └── GDPR equivalente: dato personal ordinario — Art. 6(1) bases aplicables
│
├── DATOS PRIVADOS
│   ├── Definición: íntimos o reservados, su conocimiento solo interesa al
│   │   titular (Art. 3 Ley 1581)
│   ├── Ejemplos: historia médica detallada (sin alcanzar categoría sensible),
│   │   información financiera detallada, comunicaciones privadas, datos
│   │   biográficos, contraseñas, creencias íntimas no públicas
│   ├── Régimen: requieren autorización expresa del titular
│   └── GDPR equivalente: dato personal ordinario con mayor expectativa de
│       privacidad — Arts. 5(1)(f) y 25 (privacy by design) aplicables
│
├── DATOS SENSIBLES
│   ├── Definición legal (Art. 5 Ley 1581): aquellos que afectan la intimidad
│   │   del titular o cuyo uso indebido puede generar discriminación
│   ├── Categorías taxativas:
│   │   ├── Origen racial o étnico
│   │   ├── Orientación política
│   │   ├── Convicciones religiosas o filosóficas
│   │   ├── Pertenencia a sindicatos, organizaciones sociales, de derechos humanos
│   │   ├── Datos de salud (diagnósticos, tratamientos, historia clínica)
│   │   ├── Vida sexual y orientación sexual
│   │   └── Datos biométricos (con propósito de identificación única)
│   ├── Régimen especial (Art. 6 Ley 1581):
│   │   ├── Prohibición general de tratamiento
│   │   ├── Excepción: autorización expresa del titular + finalidad legítima
│   │   │   y explícita
│   │   └── Casos de licitud sin autorización:
│   │       ├── Actividades autorizadas por ley (ej.: autoridades sanitarias)
│   │       ├── Garantizar derechos fundamentales del titular (urgencias médicas)
│   │       └── Actividades de salud pública y emergencias
│   └── GDPR equivalente (Art. 9): categorías especiales — igual prohibición
│       general con excepciones taxativas en Art. 9(2)(a-j)
│
└── DATOS DE MENORES DE EDAD
    ├── Definición: datos de personas menores de 18 años (Colombia)
    │   / menores de 16 años (GDPR Art. 8, con posibilidad de bajar a 13)
    ├── Régimen especial (Art. 7 Ley 1581):
    │   ├── Prohibición general de tratamiento
    │   ├── Excepción única: que responda y respete el interés superior del menor
    │   ├── Que se asegure el respeto de sus derechos fundamentales
    │   └── Autorización: representante legal debe otorgar consentimiento
    │       + verificación de que no va en contra del menor
    ├── Indicadores de riesgo evaluados:
    │   ├── ¿El servicio o producto puede ser accesible a menores?
    │   ├── ¿Existe verificación de edad al recolectar datos?
    │   ├── ¿Se recolectan datos de menores como parte de datos familiares?
    │   └── ¿Los datos recolectados incluyen categorías sensibles de menores?
    └── GDPR adicional: Art. 8 y considerando 38 — protección reforzada,
        no perfilado automatizado, no publicidad comportamental dirigida
```

#### 1.2.2 Algoritmo de Clasificación Automática

El sistema implementa un clasificador híbrido de tres etapas:

**Etapa 1 — Reglas Deterministas (Alta Precisión):**
```python
REGLAS_SENSIBLES = {
    "salud": ["diagnóstico", "tratamiento", "medicamento", "enfermedad", "patología",
              "historia clínica", "eps", "ips", "incapacidad", "discapacidad"],
    "biometrico": ["huella", "dactilar", "facial", "iris", "voz", "ADN", "retina"],
    "etnico_racial": ["raza", "etnia", "afrodescendiente", "indígena", "rom", "raizal"],
    "sexual": ["orientación sexual", "vida sexual", "identidad de género"],
    "politico": ["partido político", "militancia", "voto", "ideología política"],
    "religioso": ["religión", "fe", "creencia religiosa", "culto"],
    "sindical": ["sindicato", "afiliación sindical", "organización social"]
}
```

**Etapa 2 — Clasificador NLP (Modelos de Lenguaje):**
- Modelo: fine-tuned BERT/RoBERTa en corpus jurídico colombiano
- Umbral de confianza: ≥ 0.85 para clasificación autónoma
- Por debajo del umbral: escalación a revisión humana (DPO)

**Etapa 3 — Validación de Contexto:**
- Un campo "teléfono" es dato semi-privado en contexto laboral y privado en contexto personal
- La clasificación final combina tipo de dato + contexto de tratamiento + finalidad

#### 1.2.3 Requisitos de Autorización por Categoría

| Categoría | Forma de Autorización | Prueba Requerida | Revocación |
|---|---|---|---|
| Datos Públicos | No aplica (Art. 10 Ley 1581) | N/A | N/A |
| Datos Semi-privados | Escrita o verbal verificable | Grabación, clic registrado, firma | Inmediata |
| Datos Privados | Escrita preferiblemente | Documento firmado, registro digital | Inmediata |
| Datos Sensibles | Escrita, expresa, previa e informada | Formulario específico, doble confirmación | Inmediata + notificación |
| Datos de Menores | Representante legal, escrita | Documento identidad adulto + autorización | Inmediata + notificación al DPO |

---

### 1.3 Mapeo de Bases Jurídicas GDPR (Arts. 6 y 9)

#### 1.3.1 Bases de Licitud — Artículo 6 GDPR

| Base Legal (Art. 6) | Código | Descripción | Condiciones de Aplicación | Equivalente Ley 1581 |
|---|---|---|---|---|
| Consentimiento | 6(1)(a) | Titular dio consentimiento libre, específico, informado e inequívoco | Retirable en cualquier momento; no puede ser condicionado a servicio | Art. 9 — Autorización |
| Ejecución de contrato | 6(1)(b) | Tratamiento necesario para ejecutar contrato del que el titular es parte | Estricta necesidad; no tratamientos adicionales "convenientes" | Art. 10(c) |
| Obligación legal | 6(1)(c) | Tratamiento necesario para cumplir obligación legal del responsable | La ley debe ser clara y previsible; el tratamiento debe ser proporcional | Art. 10(a) |
| Intereses vitales | 6(1)(d) | Protección de intereses vitales del titular u otra persona | Solo cuando el titular no puede dar consentimiento por incapacidad | Art. 10(d) parcial |
| Interés público | 6(1)(e) | Tratamiento necesario para misión de interés público o ejercicio de poderes | Requiere base en ley de la UE/Estado miembro | Art. 10(a) parcial |
| Interés legítimo | 6(1)(f) | Intereses legítimos del responsable o tercero, salvo derechos fundamentales | Test LIA obligatorio; no aplica a autoridades públicas; no aplica a menores | No tiene equivalente directo en Ley 1581 |

#### 1.3.2 Bases de Licitud para Categorías Especiales — Artículo 9 GDPR

| Base Legal (Art. 9(2)) | Descripción | Documentación Requerida en el Sistema |
|---|---|---|
| 9(2)(a) — Consentimiento explícito | Consentimiento explícito del interesado | Formulario diferenciado, doble opt-in, granularidad por categoría |
| 9(2)(b) — Obligaciones laborales | Cumplimiento de obligaciones en ámbito laboral | Convenio colectivo, normativa laboral aplicable |
| 9(2)(c) — Intereses vitales | Proteger intereses vitales cuando es incapaz de dar consentimiento | Protocolo médico de emergencia documentado |
| 9(2)(d) — Entidades sin fines de lucro | Miembros o exmiembros de entidades religiosas, políticas, sindicales | Estatutos de la entidad, membresía documentada |
| 9(2)(e) — Datos manifiestamente públicos | El interesado los hizo públicos manifiestamente | Registro de la fuente pública y fecha |
| 9(2)(f) — Proceso judicial | Formulación, ejercicio o defensa de reclamaciones judiciales | Referencia al proceso legal, custodio de datos |
| 9(2)(g) — Interés público sustancial | Previsto en ley de la UE/Estado miembro, proporcional | Norma habilitante específica |
| 9(2)(h) — Medicina preventiva/laboral | Prestación de atención sanitaria o servicios sociales | Secreto profesional del personal sanitario |
| 9(2)(i) — Interés público en salud pública | Razones de interés público en el ámbito de la salud pública | Marco legal de salud pública aplicable |
| 9(2)(j) — Archivo, investigación, estadística | Con garantías adecuadas de los derechos del interesado | Protocolo de anonimización, medidas técnicas |

---

### 1.4 Mapeo de Controles ISO/IEC 27701:2019

ISO 27701 extiende ISO 27001/27002 añadiendo controles específicos de privacidad. El sistema mapea los controles relevantes a dimensiones auditables:

#### 1.4.1 Controles para Responsables del Tratamiento (PII Controllers)

| Control ISO 27701 | Referencia | Descripción del Control | Dimensión de Evaluación en el Sistema |
|---|---|---|---|
| 7.2.1 | Identificar y documentar la finalidad | El responsable debe documentar las finalidades antes del tratamiento | Inventario de finalidades + versionado |
| 7.2.2 | Identificar la base legal | Base jurídica documentada y válida para cada finalidad | Matriz base jurídica × finalidad |
| 7.2.3 | Determinar cuándo se requiere consentimiento | Análisis de necesidad de consentimiento vs. otras bases | Árbol de decisión de base legal |
| 7.2.4 | Obtener y registrar el consentimiento | Mecanismos técnicos de obtención y almacenamiento | Audit trail de consentimientos |
| 7.2.5 | Evaluación de impacto en privacidad (PIA) | DPIA cuando el tratamiento presenta alto riesgo | Módulo PIA/DPIA integrado |
| 7.2.6 | Contratos con encargados del tratamiento | DPA firmados con todos los encargados | Registro de DPAs |
| 7.2.7 | Registros relacionados con el tratamiento de PII | Registros de actividades de tratamiento | RNBD + Art. 30 GDPR |
| 7.3.1 | Derechos del titular: acceso | Procedimiento y canales para solicitudes de acceso | SLA de 10 días hábiles (Col) / 30 días (GDPR) |
| 7.3.2 | Derechos del titular: rectificación | Procedimiento de corrección de datos incorrectos | Workflow de rectificación |
| 7.3.3 | Derechos del titular: eliminación | Procedimiento de borrado (derecho al olvido) | Workflow de supresión |
| 7.3.4 | Derechos del titular: portabilidad | Exportación en formato estructurado y legible por máquina | API de exportación JSON/CSV |
| 7.3.5 | Derechos del titular: objeción | Canal para oposición al tratamiento | Workflow de oposición + revisión DPO |
| 7.4.1 | Notificación a titulares | Aviso de privacidad antes o en el momento de la recolección | Auditoría de avisos de privacidad |
| 7.4.2 | Proporcionar información sobre transferencias | Informar sobre transferencias internacionales | Sección de transferencias en política |
| 7.5 | Automatización del tratamiento de PII | Controles para tratamiento automatizado | Sección de auditoría de IA |

#### 1.4.2 Controles para Encargados del Tratamiento (PII Processors)

| Control ISO 27701 | Descripción | Evidencia Requerida |
|---|---|---|
| 8.2.1 | Acuerdo con el responsable | DPA firmado, cláusulas mínimas de Art. 28 GDPR | Copia del DPA cargada al sistema |
| 8.2.2 | Finalidad del tratamiento de los encargados | El encargado solo trata conforme instrucciones del responsable | Procedimiento documentado |
| 8.2.3 | Registros del encargado del tratamiento | Registros propios de actividades como encargado | Registro de actividades del encargado |
| 8.3 | Derechos del interesado | El encargado facilita el ejercicio de derechos | SLA de reenvío al responsable |
| 8.4 | Privacidad en relaciones con el cliente | Términos de servicio conformes con la normativa | Revisión de ToS |
| 8.5 | Transferencias de PII | El encargado no transfiere sin autorización del responsable | Control de sub-encargados |

---

## 2. Motor de Responsabilidad Demostrada (Accountability)

### 2.1 Diseño del Módulo de Accountability

La responsabilidad demostrada (accountability) es el principio que obliga a los responsables y encargados a implementar medidas apropiadas y efectivas para garantizar el cumplimiento normativo, y a ser capaces de demostrarlo ante las autoridades. El sistema evalúa este principio en cinco dimensiones principales.

#### 2.1.1 Arquitectura del Motor de Evaluación

```
┌─────────────────────────────────────────────────────────────────┐
│                    MOTOR DE ACCOUNTABILITY                       │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Recolección │  │   Motor de   │  │    Generación de     │  │
│  │  de Evidencia│→ │  Evaluación  │→ │    Score Global      │  │
│  │   (Upload)   │  │  (Rúbricas)  │  │   (0-100 por dim.)   │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│         ↑                  ↑                      ↓             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │    RAG AI    │  │  Factores de │  │  Plan de Remediación  │  │
│  │   (análisis  │  │  Ponderación │  │   (Tareas JIRA-like)  │  │
│  │   semántico) │  │  Contextuales│  │                      │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Factores de Ponderación Contextuales

La evaluación no aplica la misma vara a todas las organizaciones. El sistema ajusta los umbrales y pesos según:

#### 2.2.1 Naturaleza Jurídica de la Organización

| Tipo de Organización | Multiplicador | Justificación |
|---|---|---|
| Entidad pública nacional | 1.5x | Mayor escrutinio — maneja datos de toda la ciudadanía |
| Entidad pública territorial | 1.3x | Escrutinio alto — datos de ciudadanos de su jurisdicción |
| Empresa privada regulada (financiero, salud) | 1.4x | Regulaciones sectoriales adicionales (SFC, SuperSalud) |
| Empresa privada grande (>500 empleados) | 1.2x | Capacidad de cumplimiento proporcional |
| Empresa privada mediana (50-500 empleados) | 1.0x | Base de referencia |
| Empresa privada pequeña (<50 empleados) | 0.8x | Reconocimiento de limitaciones de recursos |
| Startup / Emprendimiento (<2 años) | 0.7x | Período de gracia con plan de madurez |
| ONG / Fundación | 0.9x | Propósito social, recursos limitados |

#### 2.2.2 Nivel de Riesgo del Tratamiento de Datos

El nivel de riesgo se calcula automáticamente combinando:

```
RIESGO_BASE = f(categoría_dato, volumen, finalidad, tecnología)

Matriz de Riesgo:
┌─────────────────────┬────────────┬─────────────┬─────────────┐
│                     │ <1.000     │ 1.000-      │ >100.000    │
│ Categoría / Volumen │ titulares  │ 100.000     │ titulares   │
├─────────────────────┼────────────┼─────────────┼─────────────┤
│ Datos sensibles     │ ALTO       │ MUY ALTO    │ CRÍTICO     │
│ Datos de menores    │ ALTO       │ MUY ALTO    │ CRÍTICO     │
│ Datos privados      │ MEDIO      │ ALTO        │ MUY ALTO    │
│ Datos semi-privados │ BAJO       │ MEDIO       │ ALTO        │
│ Datos públicos      │ MUY BAJO   │ BAJO        │ MEDIO       │
└─────────────────────┴────────────┴─────────────┴─────────────┘

Factores adicionales de riesgo (+1 nivel cada uno):
- Tratamiento automatizado para toma de decisiones
- Perfilado sistemático
- Uso de tecnologías de seguimiento/vigilancia
- Transferencias internacionales
- Datos de poblaciones vulnerables (adultos mayores, personas con discapacidad)
```

### 2.3 Rúbrica de Evaluación — Score 0-100 por Dimensión

#### 2.3.1 Dimensión A: Gobernanza y Estructura Organizacional (Peso: 20%)

| Indicador | Puntaje Máximo | Criterios de Evaluación |
|---|---|---|
| A1 — Política de Privacidad vigente | 15 | Publicada(5), completa Art.15 Ley1581(5), actualizada últimos 12 meses(5) |
| A2 — Responsable de Privacidad designado | 20 | Designado formalmente(10), con funciones documentadas(5), con recursos(5) |
| A3 — Programa de capacitación | 15 | Existe programa(5), anual como mínimo(5), registros de asistencia(5) |
| A4 — Registro de actividades de tratamiento | 20 | RNBD actualizado(10), inventario interno completo(10) |
| A5 — Contratos con encargados (DPAs) | 15 | 100% encargados con DPA(10), cláusulas mínimas Art.28 GDPR(5) |
| A6 — Canal de atención a titulares | 15 | Canal habilitado(5), SLA documentado(5), registro de solicitudes(5) |
| **TOTAL DIMENSIÓN A** | **100** | |

#### 2.3.2 Dimensión B: Bases Jurídicas y Autorizaciones (Peso: 25%)

| Indicador | Puntaje Máximo | Criterios de Evaluación |
|---|---|---|
| B1 — Base jurídica por finalidad documentada | 20 | 100% finalidades con base(20), 80%(15), 60%(10), <60%(0) |
| B2 — Mecanismos de obtención de autorización | 20 | Escritos y verificables(20), digitales con registro(15), verbal sin registro(5) |
| B3 — Almacenamiento de pruebas de autorización | 20 | Sistema seguro con búsqueda(20), archivo físico organizado(10), sin sistema(0) |
| B4 — Mecanismo de revocación operativo | 20 | Proceso automatizado(20), proceso manual <24h(15), proceso manual >24h(5) |
| B5 — Renovación de consentimiento | 20 | Proceso automático por cambios(20), proceso ad hoc(10), no existe(0) |
| **TOTAL DIMENSIÓN B** | **100** | |

#### 2.3.3 Dimensión C: Seguridad de la Información (Peso: 25%)

| Indicador | Puntaje Máximo | Criterios de Evaluación |
|---|---|---|
| C1 — Cifrado de datos en reposo | 15 | AES-256 implementado(15), otro cifrado fuerte(10), sin cifrado(0) |
| C2 — Cifrado de datos en tránsito | 15 | TLS 1.3(15), TLS 1.2(10), sin TLS(0) |
| C3 — Control de acceso (RBAC/ABAC) | 15 | Granular con MFA(15), básico con roles(10), sin control(0) |
| C4 — Logs de auditoría | 15 — | Inmutables, centralizados, 12 meses(15), parciales(8), sin logs(0) |
| C5 — Gestión de incidentes | 15 | Proceso documentado y probado(15), documentado sin prueba(8), no existe(0) |
| C6 — Evaluación de vulnerabilidades | 10 | Pentesting anual(10), escaneo automatizado(6), sin evaluación(0) |
| C7 — Plan de continuidad | 15 | BCP/DRP documentado y probado(15), documentado sin prueba(8), no existe(0) |
| **TOTAL DIMENSIÓN C** | **100** | |

#### 2.3.4 Dimensión D: Derechos de los Titulares (Peso: 15%)

| Indicador | Puntaje Máximo | Criterios de Evaluación |
|---|---|---|
| D1 — Canal de solicitudes de derechos | 20 | Digital automatizado(20), correo con respuesta(12), no existe(0) |
| D2 — Cumplimiento de SLA legal | 25 | 100% en tiempo(25), 80%(18), 60%(10), <60%(0) |
| D3 — Registro de solicitudes y respuestas | 20 | Sistema completo con evidencias(20), registro parcial(10), sin registro(0) |
| D4 — Portabilidad de datos | 15 | JSON/CSV exportable(15), exportación manual(8), no existe(0) |
| D5 — Proceso de eliminación / derecho al olvido | 20 | Proceso técnico verificado(20), proceso manual(12), no existe(0) |
| **TOTAL DIMENSIÓN D** | **100** | |

#### 2.3.5 Dimensión E: Transferencias y Terceros (Peso: 15%)

| Indicador | Puntaje Máximo | Criterios de Evaluación |
|---|---|---|
| E1 — Inventario de encargados/terceros | 25 | Completo y actualizado(25), parcial(12), no existe(0) |
| E2 — DPAs firmados con encargados | 25 | 100% con DPA(25), 80%(18), 60%(10), <60%(0) |
| E3 — Transferencias internacionales | 25 | Garantías documentadas por país(25), parcial(12), sin controles(0) |
| E4 — Auditoría de encargados | 25 | Anual documentada(25), ad hoc(12), sin auditoría(0) |
| **TOTAL DIMENSIÓN E** | **100** | |

#### 2.3.6 Fórmula de Score Global

```
SCORE_GLOBAL = (A × 0.20) + (B × 0.25) + (C × 0.25) + (D × 0.15) + (E × 0.15)

Interpretación del Score:
┌─────────────────┬──────────────────┬─────────────────────────────────────────┐
│ Rango de Score  │ Nivel            │ Descripción y Acción Requerida          │
├─────────────────┼──────────────────┼─────────────────────────────────────────┤
│ 85 - 100        │ CONFORME         │ Cumplimiento sustancial. Mejora continua │
│ 70 - 84         │ CONFORME PARCIAL │ Brechas menores. Plan de mejora 90 días │
│ 50 - 69         │ NO CONFORME      │ Brechas significativas. Plan 60 días     │
│ 30 - 49         │ RIESGO ALTO      │ Incumplimiento grave. Plan inmediato     │
│ 0 - 29          │ CRÍTICO          │ Riesgo de sanción. Asesoría urgente      │
└─────────────────┴──────────────────┴─────────────────────────────────────────┘
```

---

### 2.4 Gestión del Registro Nacional de Bases de Datos (RNBD)

El RNBD es el repositorio administrado por la SIC donde los responsables del tratamiento deben inscribir sus bases de datos de datos personales (Decreto 090/2018).

#### 2.4.1 Flujo de Registro Inicial

```
FLUJO DE REGISTRO RNBD
        │
        ▼
[1. Identificación de Bases de Datos]
   - Inventario automático via IA (análisis de sistemas del tenant)
   - Clasificación por categoría de datos
   - Estimación de volumen de titulares
        │
        ▼
[2. Verificación de Obligación de Registro]
   - ¿Trata datos de ciudadanos colombianos? → SÍ: obliga registro
   - ¿Es responsable del tratamiento? → SÍ: obliga registro
   - ¿Base de datos contiene datos sensibles? → SÍ: prioridad alta
        │
        ▼
[3. Preparación de Información Requerida]
   Campos según Decreto 090/2018:
   ├── Identificación del responsable (NIT, razón social, dirección)
   ├── Nombre de la base de datos
   ├── Finalidad(es) del tratamiento
   ├── Política de tratamiento (URL o documento)
   ├── Categoría de los datos (pública, semiprivada, privada, sensible)
   ├── Mecanismos de seguridad implementados
   ├── Transferencias internacionales (si/no + países)
   ├── Fecha de creación de la base de datos
   └── Datos del Oficial de Privacidad (si lo tiene)
        │
        ▼
[4. Registro en Portal SIC]
   - Asistente guiado en la plataforma
   - Validación de campos antes de envío
   - Generación de reporte de registro para archivo
        │
        ▼
[5. Confirmación y Monitoreo]
   - Almacenamiento del número de registro
   - Configuración de alertas de vencimiento (renovación anual)
   - Dashboard de estado de cada base registrada
```

#### 2.4.2 Flujo de Actualización del RNBD

El RNBD debe actualizarse cuando ocurran cambios materiales:

| Evento Disparador | Plazo para Actualización | Acción en el Sistema |
|---|---|---|
| Cambio de finalidad del tratamiento | 10 días hábiles | Alerta automática + workflow de actualización |
| Nueva base de datos creada | 10 días hábiles | Detección automática + solicitud de registro |
| Cambio de responsable del tratamiento | 10 días hábiles | Alerta + actualización de datos de contacto |
| Cambio en mecanismos de seguridad | 10 días hábiles | Actualización de sección de seguridad |
| Inicio de transferencias internacionales | Antes de iniciar | Evaluación de garantías + actualización |
| Cese de tratamiento de una base | 10 días hábiles | Marcado para supresión del registro |

#### 2.4.3 Flujo de Eliminación del RNBD

```
Evento: Cese de tratamiento de base de datos
        │
        ▼
[Verificación previa]
├── ¿Existen obligaciones legales de retención que impidan la eliminación?
│   (ej.: contabilidad 10 años, laboral 5 años, tributario 5 años)
│   ├── SÍ: Mover a "Retención Legal" — no eliminar del RNBD aún
│   └── NO: Proceder con eliminación
        │
        ▼
[Proceso de eliminación seguro]
├── Anonimización o destrucción de datos según estándar DoD 5220.22-M
├── Evidencia de destrucción (certificado)
├── Notificación a encargados para eliminación en sus sistemas
        │
        ▼
[Actualización del RNBD en la SIC]
├── Marcado de la base como "Suprimida" en el RNBD
├── Fecha de supresión y motivo
└── Archivo de la documentación de destrucción
```

---

### 2.5 Evaluación y Designación del Oficial de Privacidad (DPO)

#### 2.5.1 Criterios de Obligatoriedad

| Criterio | Fuente Normativa | Umbral |
|---|---|---|
| Monitoreo sistemático a gran escala | GDPR Art. 37(1)(b) | Perfilado regular y sistemático de individuos |
| Categorías especiales a gran escala | GDPR Art. 37(1)(c) | Tratamiento como actividad principal |
| Autoridad u organismo público | GDPR Art. 37(1)(a) | Siempre obligatorio (excepto tribunales) |
| Recomendación SIC Colombia | Circular SIC | Toda empresa que trate datos sensibles o de >10.000 titulares |

El sistema genera un reporte de obligatoriedad de DPO basado en las características del tenant.

#### 2.5.2 Rúbrica de Evaluación del DPO

```
EVALUACIÓN DEL OFICIAL DE PRIVACIDAD (0-100)

Criterio 1: Independencia (25 puntos)
├── ¿El DPO reporta directamente a la alta dirección? (10)
├── ¿Puede emitir opiniones sin instrucciones sobre cómo actuar? (10)
└── ¿No tiene conflicto de interés con sus otras funciones? (5)

Criterio 2: Conocimiento (25 puntos)
├── ¿Tiene formación certificada en protección de datos? (10)
│   [CIPP/E, CIPP/A, CIPM, certificación SIC Colombia]
├── ¿Conoce el sector de actividad de la empresa? (8)
└── ¿Se actualiza regularmente (al menos 20h/año de formación)? (7)

Criterio 3: Recursos (25 puntos)
├── ¿Tiene tiempo suficiente para sus funciones (no es un cargo secundario)? (10)
├── ¿Tiene presupuesto asignado? (8)
└── ¿Tiene acceso a sistemas y documentación relevante? (7)

Criterio 4: Funciones Documentadas (25 puntos)
├── ¿Tiene funciones formalmente asignadas por escrito? (8)
├── ¿Participa en PIA/DPIA? (8)
└── ¿Es el punto de contacto con la autoridad de control? (9)
```

---

### 2.6 Evaluación de Impacto en la Privacidad (PIA/DPIA)

#### 2.6.1 Criterios de Obligatoriedad de DPIA

Según el Art. 35 GDPR, una DPIA es obligatoria cuando el tratamiento es "susceptible de entrañar un alto riesgo". El sistema evalúa nueve criterios del EDPB (WP 248):

| Criterio WP248 | Descripción | Detección Automática |
|---|---|---|
| C1 — Evaluación o puntuación | Perfilado, scoring crediticio, predicciones | Análisis de finalidades declaradas |
| C2 — Decisión automatizada con efectos jurídicos | Aprobación/denegación automatizada | Módulo de auditoría de IA |
| C3 — Vigilancia sistemática | CCTV, monitoreo de empleados, tracking | Inventario de tecnologías |
| C4 — Datos sensibles o de alto riesgo | Art. 9 GDPR / Art. 5 Ley 1581 | Clasificación de datos |
| C5 — Datos a gran escala | >250.000 titulares o >1% población | Volumen de bases de datos |
| C6 — Cruces de conjuntos de datos | Combinación de múltiples fuentes | Análisis de flujos de datos |
| C7 — Datos sobre personas vulnerables | Menores, empleados, pacientes, ancianos | Categorización de titulares |
| C8 — Uso innovador de tecnología | Biometría, IA generativa, IoT | Inventario tecnológico |
| C9 — Transferencia fuera de EEE/Colombia | Países sin nivel adecuado | Módulo de transferencias |

**Regla:** ≥ 2 criterios presentes → DPIA obligatoria.

#### 2.6.2 Estructura de la DPIA en el Sistema

```yaml
dpia_estructura:
  seccion_1_descripcion:
    - naturaleza_tratamiento
    - alcance_y_contexto
    - finalidades
    - datos_procesados
    - actores_involucrados
    
  seccion_2_necesidad_proporcionalidad:
    - finalidades_legitimas_especificadas
    - duracion_minima_necesaria
    - datos_minimos_necesarios
    - base_legal_aplicable
    - calidad_de_los_datos
    - derechos_informados_y_ejercibles
    
  seccion_3_gestion_de_riesgos:
    - identificacion_de_riesgos:
        - acceso_no_autorizado
        - modificacion_no_deseada
        - desaparicion_de_datos
    - nivel_de_riesgo_inherente: [insignificante, limitado, importante, maximo]
    - medidas_mitigadoras
    - nivel_de_riesgo_residual
    
  seccion_4_aprobacion:
    - opinion_del_dpo
    - aceptacion_del_responsable
    - fecha_de_revision_siguiente
    - estado: [borrador, en_revision, aprobada, rechazada]
```

---

## 3. Gobernanza de Flujos Transfronterizos y Tecnológicos

### 3.1 Motor de Alertas para Transferencias Internacionales

Una transferencia internacional ocurre cuando los datos personales se comunican, envían, transmiten o transfieren a un receptor ubicado en otro país. El sistema implementa un motor de alertas de tres niveles.

#### 3.1.1 Arquitectura del Motor de Detección

```
FUENTES DE DETECCIÓN DE TRANSFERENCIAS
├── Declaración explícita del tenant (formulario de inventario)
├── Análisis de contratos con proveedores (NLP sobre documentos cargados)
├── Detección de IPs de servidores en el inventario tecnológico
└── Análisis de flujos de integración API declarados

                         ↓
            MOTOR DE CLASIFICACIÓN POR PAÍS
                         ↓
┌─────────────────────────────────────────────────────┐
│  ¿País destino tiene nivel de protección adecuado?  │
│                                                      │
│  ADECUADO → VERDE (transferencia libre)             │
│  NO ADECUADO + garantías → AMARILLO (verificar)     │
│  NO ADECUADO + sin garantías → ROJO (bloquear)      │
└─────────────────────────────────────────────────────┘
```

#### 3.1.2 Lista de Países con Nivel de Protección Adecuado

**Decisiones de Adecuación de la Comisión Europea (vigentes a 2026):**

| País / Territorio | Decisión | Vigencia | Tipo de Datos | Observaciones |
|---|---|---|---|---|
| Andorra | 2010/625/UE | Indefinida | Todos | Revisión periódica |
| Argentina | 2003/490/CE | Indefinida | Todos | Bajo revisión por EDPB |
| Canadá (sector privado) | 2002/2/CE | Indefinida | Sector privado — PIPEDA | No aplica a sector público |
| Islas Feroe | 2010/146/UE | Indefinida | Todos | |
| Guernsey | 2003/821/CE | Indefinida | Todos | |
| Israel | 2011/61/UE | Indefinida | Todos | |
| Isla de Man | 2004/411/CE | Indefinida | Todos | |
| Japón | C(2019)61 | Indefinida | Todos | Marco recíproco con UE |
| Jersey | 2008/393/CE | Indefinida | Todos | |
| Nueva Zelanda | 2013/65/UE | Indefinida | Todos | |
| República de Corea | C(2021)4800 | Indefinida | Todos | |
| Suiza | 2000/518/CE | Indefinida | Todos | En revisión tras nFADP 2023 |
| Uruguay | 2012/484/UE | Indefinida | Todos | |
| Reino Unido | C(2021)4800 | Indefinida | Todos | Post-Brexit — sujeta a revisión |
| EE.UU. (DPF) | C(2023)4745 | Indefinida | Participantes en DPF | Data Privacy Framework |

**Nota Colombia:** La Ley 1581 no tiene un sistema de adecuación equivalente. Para transferencias desde Colombia, se requiere declaración de conformidad con el Art. 26 (garantías adecuadas o consentimiento explícito).

#### 3.1.3 Países SIN Adecuación — Mecanismos Alternativos Requeridos

```
ÁRBOL DE DECISIÓN — TRANSFERENCIA A PAÍS SIN ADECUACIÓN

¿El responsable tiene SCCs firmadas con el importador?
├── SÍ → ¿Se realizó TIA (Transfer Impact Assessment)?
│         ├── SÍ → ¿Hay medidas suplementarias si el TIA indica riesgo?
│         │         ├── SÍ → AMARILLO — Permitido con monitoreo
│         │         └── NO → ROJO — Suspender hasta implementar medidas
│         └── NO → ADVERTENCIA — Completar TIA antes de continuar
└── NO → ¿Existen BCR aprobadas?
           ├── SÍ → AMARILLO — Verificar vigencia de aprobación
           └── NO → ¿Existe excepción aplicable del Art. 49 GDPR?
                       ├── Consentimiento explícito informado del riesgo → Caso a caso
                       ├── Ejecución de contrato necesario → Verificar necesidad
                       ├── Intereses vitales → Solo para urgencias médicas
                       └── NINGUNA → ROJO — Transferencia prohibida
```

---

### 3.2 Privacy by Design en Transferencias Tecnológicas

El módulo evalúa si las transferencias implementan el enfoque de Privacidad desde el Diseño (PbD), desarrollado por Ann Cavoukian y consagrado en el Art. 25 GDPR.

#### 3.2.1 Los 7 Principios de PbD Evaluados

| Principio PbD | Descripción | Indicador en el Sistema | Puntaje |
|---|---|---|---|
| 1. Proactivo, no reactivo | Prevenir problemas antes de que ocurran | ¿Se realizó PIA antes de implementar la transferencia? | 15 |
| 2. Privacidad como configuración predeterminada | La opción más protectora es el valor por defecto | ¿La configuración por defecto limita al máximo la transferencia? | 15 |
| 3. Privacidad integrada en el diseño | No un añadido sino parte de la arquitectura | ¿Los controles de privacidad son arquitectónicos, no post-hoc? | 15 |
| 4. Funcionalidad plena (win-win) | Privacidad sin sacrificar funcionalidad | ¿La privacidad se logró sin degradar el servicio? | 10 |
| 5. Seguridad de principio a fin | Protección durante todo el ciclo de vida | ¿Los datos están cifrados en origen, tránsito y destino? | 15 |
| 6. Visibilidad y transparencia | Las prácticas son verificables | ¿El titular puede verificar que la transferencia ocurre y por qué? | 15 |
| 7. Respeto por la privacidad del usuario | Centrado en el titular, no en el negocio | ¿El titular tiene control real sobre sus datos en el país destino? | 15 |
| **TOTAL** | | | **100** |

---

### 3.3 Módulo de Evaluación de Cláusulas Contractuales Tipo (SCCs)

#### 3.3.1 Versiones de SCCs Evaluadas por el Sistema

| Versión | Autoridad | Estado | Módulos |
|---|---|---|---|
| SCCs UE 2021 (C(2021)914) | Comisión Europea | VIGENTE — Obligatorio desde 27/12/2022 | Módulo 1: C2C / Módulo 2: C2P / Módulo 3: P2P / Módulo 4: P2C |
| SCCs antiguas 2001/2004 | Comisión Europea | DEROGADAS — No válidas para nuevos contratos | — |
| SCCs UK IDTA | ICO (UK) | VIGENTE para transferencias desde UK | Adenda al contrato existente |
| Cláusulas Modelo Colombia | SIC | RECOMENDADAS — No obligatorias aún | Basadas en SCCs UE |

#### 3.3.2 Checklist de Evaluación de SCCs

```
EVALUACIÓN DE SCCs (Módulo 2 — Controller to Processor)

Sección 1 — Cláusulas Generales
□ ¿Se especifican las partes correctamente (exportador e importador)?
□ ¿Se describe el tratamiento objeto de la transferencia?
□ ¿Se especifican los derechos de los interesados?
□ ¿Se establece la ley aplicable?

Sección 2 — Obligaciones del Exportador
□ ¿El exportador ha verificado que las SCCs se firmaron correctamente?
□ ¿El exportador realizó TIA antes de la transferencia?
□ ¿El exportador tiene instrucciones documentadas al importador?

Sección 3 — Obligaciones del Importador
□ ¿El importador acepta la supervisión del exportador?
□ ¿El importador notificará al exportador de solicitudes de autoridades?
□ ¿El importador ha documentado las medidas técnicas de protección?

Sección 4 — Sub-encargados
□ ¿El importador lista todos sus sub-encargados en el Anexo III?
□ ¿El exportador autorizó el uso de sub-encargados?
□ ¿Los sub-encargados tienen obligaciones equivalentes?

Anexos
□ Anexo I: Descripción del tratamiento completamente cumplimentado
□ Anexo II: Medidas técnicas y organizativas (TOMs) específicas
□ Anexo III: Lista de sub-encargados autorizada
```

---

### 3.4 Evaluación de Binding Corporate Rules (BCR)

Las BCR son políticas de protección de datos vinculantes para grupos empresariales multinacionales, aprobadas por la autoridad de control líder bajo el Art. 47 GDPR.

#### 3.4.1 Verificación de BCR en el Sistema

```json
{
  "evaluacion_bcr": {
    "verificacion_existencia": {
      "pregunta": "¿El grupo empresarial cuenta con BCR aprobadas?",
      "fuente_verificacion": "Registro EDPB de BCR aprobadas (público)",
      "url": "https://www.edpb.europa.eu/our-work-tools/accountability-tools/bcr_es"
    },
    "verificacion_vigencia": {
      "pregunta": "¿Las BCR están vigentes (no suspendidas ni revocadas)?",
      "alerta": "Verificar si la autoridad líder ha emitido medidas correctivas recientes"
    },
    "verificacion_cobertura": {
      "pregunta": "¿Las BCR cubren al exportador Y al importador de los datos?",
      "riesgo": "Las BCR solo cubren entidades del mismo grupo — no aplican a terceros"
    },
    "verificacion_actualidad": {
      "pregunta": "¿Las BCR se actualizaron después de Schrems II (julio 2020)?",
      "razon": "Schrems II impuso requisitos adicionales de evaluación de acceso gubernamental"
    }
  }
}
```

---

## 4. Auditoría de IA y Algoritmos

### 4.1 Marco de Evaluación de Sistemas de IA

El sistema evalúa los sistemas de inteligencia artificial que las empresas cliente utilizan para tratar datos personales, aplicando un marco que combina criterios de la Ley de IA de la UE (AI Act), el GDPR y los principios de la Ley 1581.

#### 4.1.1 Clasificación de Riesgo de Sistemas de IA

```
CLASIFICACIÓN AI ACT (UE) — Sistema de Cuatro Niveles

NIVEL 1 — RIESGO INACEPTABLE (PROHIBIDOS)
├── Sistemas de puntuación social por autoridades públicas
├── Manipulación subliminal que cause daño
├── Explotación de vulnerabilidades de grupos específicos
└── Reconocimiento biométrico en tiempo real en espacios públicos
    (excepciones muy limitadas para seguridad nacional)

NIVEL 2 — ALTO RIESGO (REQUISITOS ESTRICTOS)
├── Infraestructuras críticas (energía, agua, transporte)
├── Educación (acceso, evaluación de estudiantes)
├── Empleo (selección, gestión de empleados)
├── Servicios esenciales (crédito, seguros, servicios públicos)
├── Aplicaciones policiales
├── Migración y asilo
└── Administración de justicia

NIVEL 3 — RIESGO LIMITADO (OBLIGACIONES DE TRANSPARENCIA)
├── Chatbots y asistentes conversacionales
├── Sistemas de generación de contenido (deepfakes)
└── Sistemas de reconocimiento de emociones

NIVEL 4 — RIESGO MÍNIMO
└── Filtros de spam, IA en videojuegos, etc.
```

#### 4.1.2 Cuatro Criterios de Evaluación Jurídica de IA

**Criterio I — Idoneidad:**
El sistema de IA debe ser adecuado y apto para alcanzar la finalidad legítima perseguida.

```
Evaluación de Idoneidad (0-25 puntos):
├── ¿El sistema de IA alcanza efectivamente la finalidad declarada? (8)
│   Evidencia: métricas de desempeño documentadas, validación independiente
├── ¿La finalidad es legítima y reconocida por el ordenamiento jurídico? (9)
│   Evidencia: base jurídica del tratamiento, análisis de legitimidad
└── ¿Hay relación causal verificable entre el uso del sistema y el logro
    de la finalidad (no correlación espuria)? (8)
    Evidencia: documentación técnica del modelo, estudios de validación
```

**Criterio II — Necesidad:**
El tratamiento de datos mediante IA debe ser necesario, es decir, no existe un medio alternativo menos intrusivo que alcance igualmente la finalidad.

```
Evaluación de Necesidad (0-25 puntos):
├── ¿Se analizaron alternativas menos invasivas de privacidad? (10)
│   Evidencia: análisis de alternativas documentado, decisión motivada
├── ¿El sistema usa el mínimo de datos necesarios para la finalidad? (8)
│   Evidencia: justificación de cada variable de entrada, análisis de
│   feature importance
└── ¿La precisión del sistema justifica el nivel de intrusión en privacidad? (7)
    Evidencia: comparativa de precisión entre sistema actual y alternativas
```

**Criterio III — Razonabilidad:**
El tratamiento no debe vulnerar los derechos y garantías de los titulares de manera desproporcionada.

```
Evaluación de Razonabilidad (0-25 puntos):
├── ¿El titular puede comprender por qué el sistema tomó una decisión? (10)
│   Evidencia: mecanismos de explicabilidad (SHAP, LIME, etc.)
├── ¿El titular puede impugnar la decisión automatizada? (8)
│   Evidencia: proceso de revisión humana documentado
└── ¿El sistema fue sometido a evaluación de impacto en derechos
    fundamentales? (7)
    Evidencia: FRIA (Fundamental Rights Impact Assessment) o DPIA
```

**Criterio IV — Proporcionalidad:**
Existe equilibrio entre los beneficios del tratamiento y los riesgos para los titulares.

```
Evaluación de Proporcionalidad (0-25 puntos):
├── ¿El beneficio para el responsable/sociedad es mayor que el riesgo
    para los titulares? (10)
    Evidencia: análisis coste-beneficio documentado, consulta a titulares
├── ¿Las medidas de mitigación de riesgo son proporcionales al nivel
    de riesgo identificado? (8)
    Evidencia: DPIA aprobada, medidas técnicas implementadas
└── ¿La duración del tratamiento es proporcional a la finalidad? (7)
    Evidencia: política de retención de datos del modelo, ciclo de vida del
    sistema
```

---

### 4.2 Decisiones Automatizadas — Evaluación GDPR Art. 22

#### 4.2.1 Identificación de Decisiones Automatizadas

El sistema evalúa si las empresas cliente utilizan procesos que constituyen "decisiones automatizadas" en el sentido del Art. 22 GDPR:

```
ÁRBOL DE DECISIÓN — ¿Es una decisión automatizada sujeta al Art. 22?

¿La decisión se basa únicamente en el tratamiento automatizado?
├── NO (hay intervención humana real, no ceremonial) → NO aplica Art. 22
└── SÍ → ¿La decisión produce efectos jurídicos o significativos?
             ├── NO (decisiones menores sin impacto real) → NO aplica Art. 22
             └── SÍ → APLICA ART. 22 — Requisitos obligatorios:
                       ├── Derecho a no ser objeto de la decisión
                       │   (salvo excepciones del Art. 22(2))
                       ├── Excepciones: contrato, ley, consentimiento explícito
                       ├── Si excepción aplica → Garantías mínimas:
                       │   ├── Intervención humana significativa
                       │   ├── Derecho a expresar punto de vista
                       │   └── Derecho a impugnar la decisión
                       └── Si usa categorías especiales → Requiere 22(4):
                           Consentimiento explícito O interés público sustancial
```

#### 4.2.2 Evaluación de Perfilado

| Dimensión de Evaluación | Pregunta | Puntaje Máximo |
|---|---|---|
| Transparencia del perfilado | ¿Los titulares son informados de que son perfilados y con qué finalidad? | 20 |
| Base jurídica del perfilado | ¿Existe base jurídica específica para el perfilado (no solo para el tratamiento base)? | 20 |
| Oposición al perfilado | ¿Existe mecanismo real de oposición al perfilado con efecto real? | 20 |
| Limitación de efectos | ¿El perfilado solo tiene efectos en decisiones no significativas? | 20 |
| Segmentación de menores | ¿Está excluido el perfilado de menores de edad? | 20 |
| **TOTAL** | | **100** |

---

### 4.3 Marco de Detección de Sesgo Algorítmico

#### 4.3.1 Tipos de Sesgo Evaluados

| Tipo de Sesgo | Descripción | Métricas de Detección |
|---|---|---|
| Sesgo de selección | Los datos de entrenamiento no representan a la población objetivo | Análisis de distribución demográfica del dataset |
| Sesgo de confirmación | El modelo refuerza patrones históricos discriminatorios | Análisis de correlaciones con variables protegidas |
| Sesgo de agregación | Usar un modelo único para grupos que requieren modelos distintos | Análisis de rendimiento por subgrupos |
| Sesgo de medición | Las variables proxy capturan atributos protegidos | Análisis de causalidad entre features y variables sensibles |
| Sesgo de evaluación | Las métricas de éxito favorecen a grupos mayoritarios | Evaluación de equidad por métricas específicas |

#### 4.3.2 Métricas de Equidad (Fairness)

```python
# Métricas de equidad evaluadas por el sistema
METRICAS_EQUIDAD = {
    "demographic_parity": {
        "descripcion": "La tasa de resultados positivos es igual en todos los grupos",
        "formula": "P(Y_hat=1|A=0) = P(Y_hat=1|A=1)",
        "umbral_aceptable": "diferencia < 0.05"
    },
    "equalized_odds": {
        "descripcion": "TPR y FPR son iguales en todos los grupos",
        "formula": "P(Y_hat=1|Y=1,A=0) = P(Y_hat=1|Y=1,A=1) AND P(Y_hat=1|Y=0,A=0) = P(Y_hat=1|Y=0,A=1)",
        "umbral_aceptable": "diferencia < 0.05 en ambas tasas"
    },
    "calibration": {
        "descripcion": "Las probabilidades predichas reflejan frecuencias reales por grupo",
        "umbral_aceptable": "ECE (Expected Calibration Error) < 0.05 por grupo"
    },
    "individual_fairness": {
        "descripcion": "Individuos similares reciben predicciones similares",
        "metodo": "Análisis de vecindad en el espacio de features"
    }
}
```

---

### 4.4 Requisitos de Transparencia y Explicabilidad

#### 4.4.1 Checklist de Transparencia Algorítmica

```
NIVEL 1 — TRANSPARENCIA BÁSICA (mínimo legal)
□ ¿Se informa al titular de que existe un sistema de IA tomando decisiones?
□ ¿Se informa la lógica general del sistema (no necesariamente el código)?
□ ¿Se informa la importancia (relevancia) de los datos usados?
□ ¿Se informa las consecuencias previsibles para el titular?

NIVEL 2 — TRANSPARENCIA FUNCIONAL (buenas prácticas)
□ ¿Existe documentación técnica del modelo (model card)?
□ ¿Se publican las métricas de rendimiento y equidad del modelo?
□ ¿Se documenta el proceso de validación y prueba?
□ ¿Se documenta el dataset de entrenamiento (data sheet)?

NIVEL 3 — EXPLICABILIDAD (alta madurez)
□ ¿El sistema puede generar explicaciones individuales? (SHAP, LIME, ANCHOR)
□ ¿Las explicaciones son comprensibles para un no-experto?
□ ¿Las explicaciones son fidedignas (no post-hoc racionalizaciones)?
□ ¿Existe revisión humana para explicaciones en casos de alto impacto?
```

#### 4.4.2 Metodología de Scoring de IA

```
SCORE DE MADUREZ EN IA RESPONSABLE (0-100)

Bloque 1: Gobernanza de IA (25%)
├── Política de IA Responsable documentada (10)
├── Responsable de IA / Comité de ética designado (8)
└── Proceso de aprobación de nuevos sistemas de IA (7)

Bloque 2: Técnico — Desarrollo (25%)
├── Análisis de sesgo en fase de diseño (8)
├── Dataset documentado (data sheet) (8)
└── Validación independiente del modelo (9)

Bloque 3: Técnico — Operación (25%)
├── Monitoreo continuo del rendimiento y sesgo en producción (10)
├── Explicabilidad individual disponible (8)
└── Proceso de actualización/retiro del modelo documentado (7)

Bloque 4: Derechos y Gobernanza (25%)
├── Proceso de impugnación de decisiones automatizadas (10)
├── Revisión humana real disponible (8)
└── Canal de reclamación por discriminación algorítmica (7)
```

---

## 5. Arquitectura de UI/UX y Multitenencia

### 5.1 Dashboards Analíticos

#### 5.1.1 Dashboard Principal — Compliance Overview

El dashboard principal presenta al usuario (según su rol) una visión centralizada del estado de cumplimiento de la organización.

**Componentes del Dashboard Principal:**

```
┌─────────────────────────────────────────────────────────────────────┐
│  COMPLIANCE OVERVIEW                          [Tenant: Empresa XYZ] │
│  Período: Junio 2026                        [Exportar PDF] [Alertas] │
├──────────────┬──────────────┬──────────────┬────────────────────────┤
│ SCORE GLOBAL │  DIMENSIÓN A │  DIMENSIÓN B │      DIMENSIÓN C       │
│    78/100    │   Gobernanza │  Autorizac.  │      Seguridad         │
│  ●●●●○○○○○○  │    82/100    │   71/100     │       85/100           │
│  Conforme    │    Verde     │   Amarillo   │       Verde            │
│  Parcial     │              │              │                        │
├──────────────┴──────────────┴──────────────┴────────────────────────┤
│  TENDENCIA ÚLTIMOS 6 MESES                                          │
│  [Gráfico de líneas: score mensual con hitos de diagnóstico]        │
│  Ene:65 | Feb:68 | Mar:72 | Abr:74 | May:77 | Jun:78               │
├─────────────────────────────────────────────────────────────────────┤
│  ALERTAS ACTIVAS (3)                                                │
│  🔴 CRÍTICO: RNBD sin actualizar (90 días) — [Ver] [Remediar]       │
│  🟡 ALTO: DPO sin certificación válida — [Ver] [Remediar]           │
│  🟡 ALTO: 2 encargados sin DPA firmado — [Ver] [Remediar]           │
├─────────────────────────────────────────────────────────────────────┤
│  PRÓXIMAS ACCIONES                    │  ESTADÍSTICAS RÁPIDAS       │
│  • Renovar política privacidad (15d)  │  Bases de datos: 12         │
│  • Completar DPIA para CRM nuevo (8d) │  Encargados: 8              │
│  • Capacitación equipo TI (3d)        │  Titulares aprox.: 45.000   │
└─────────────────────────────────────────────────────────────────────┘
```

#### 5.1.2 GeoScore — Mapa de Cumplimiento Geográfico

El GeoScore es una visualización heatmap que muestra el nivel de cumplimiento y riesgo de transferencias por país/región.

**Lógica del GeoScore:**

| Indicador Visual | Color | Significado | Acción Requerida |
|---|---|---|---|
| País sin transferencias | Gris | No aplica | Ninguna |
| Transferencia + adecuación | Verde | Conforme | Monitoreo periódico |
| Transferencia + SCCs vigentes + TIA | Amarillo | Conforme con vigilancia | Revisión anual |
| Transferencia + SCCs desactualizadas | Naranja | Riesgo moderado | Actualizar SCCs |
| Transferencia sin garantías | Rojo | Incumplimiento | Suspender o documentar |
| País en lista negra (restricciones) | Rojo oscuro | Prohibido | Cese inmediato |

**Interactividad del GeoScore:**
- Click en país: panel lateral con detalles de transferencias a ese país
- Panel incluye: volumen de datos, base jurídica, garantías, vencimiento de SCCs
- Exportar: informe de transferencias internacionales en PDF para autoridades

---

### 5.2 Sistema de Gestión de Remediación (Kanban)

El módulo de remediación transforma los hallazgos del diagnóstico en tareas accionables con seguimiento completo.

#### 5.2.1 Estructura de Tablero Kanban

```
TABLERO DE REMEDIACIÓN — Empresa XYZ

[BACKLOG]          [EN PROGRESO]       [REVISIÓN]        [COMPLETADO]
────────────       ─────────────       ──────────        ────────────
📋 Actualizar      📋 Firmar DPA       📋 Certificado     ✅ Política
   RNBD              con Proveedor       DPO — Enviado      Privacidad
   P: CRÍTICO         AWS                 a Dirección       Actualizada
   Owner: DPO        P: ALTO             P: ALTO           10/06/2026
   Due: 10 jul       Owner: Legal        Owner: RRHH
   [Evidencia]       Due: 30 jun         Due: 15 jul
                     [Upload]

📋 Completar       📋 Implementar
   DPIA para          MFA en
   Sistema CRM        sistemas
   P: ALTO            críticos
   Owner: TI          P: MEDIO
   Due: 20 jul        Owner: TI
                      Due: 31 jul
```

#### 5.2.2 Esquema de Priorización de Tareas

```
PRIORIDAD CRÍTICA (P0)
├── Criterios: riesgo de sanción inmediata, brecha de datos activa,
│   transferencia no autorizada en curso
├── SLA de inicio: 24 horas
└── Escalación: automática a Tenant Admin y DPO

PRIORIDAD ALTA (P1)
├── Criterios: incumplimiento normativo sin brecha activa, RNBD vencido,
│   DPA faltante con encargado activo
├── SLA de inicio: 3 días hábiles
└── Escalación: automática al DPO si no hay movimiento en 5 días

PRIORIDAD MEDIA (P2)
├── Criterios: mejoras de buenas prácticas, documentación incompleta,
│   formación pendiente
├── SLA de inicio: 10 días hábiles
└── Escalación: reporte mensual al Tenant Admin

PRIORIDAD BAJA (P3)
├── Criterios: optimizaciones, mejoras de UX para titulares,
│   actualizaciones de procesos menores
├── SLA de inicio: 30 días hábiles
└── Escalación: ninguna automática
```

#### 5.2.3 Sistema de Evidencias

Cada tarea de remediación puede tener evidencias adjuntas:

```yaml
evidence_types:
  documento:
    formatos: [PDF, DOCX, ODT]
    max_size: 50MB
    verificacion: hash SHA-256 almacenado para integridad
  
  imagen:
    formatos: [PNG, JPG, WEBP]
    max_size: 10MB
    uso: capturas de pantalla de configuraciones
  
  url_externa:
    tipo: enlace verificable a recurso externo
    verificacion: snapshot del recurso en fecha de carga
  
  formulario_plataforma:
    tipo: formulario completado dentro de la plataforma
    valor: evidencia nativa con timestamp y firma digital
  
  certificado:
    formatos: [PDF]
    verificacion: extraccion de metadatos (emisor, fechas, titular)
    validez: verificacion contra lista de revocaciones si aplica
```

---

### 5.3 Modelo RBAC — Control de Acceso Basado en Roles

#### 5.3.1 Jerarquía de Roles

```
SUPERADMIN (Plataforma)
│  Puede: gestionar tenants, ver auditorías de plataforma, 
│         configurar parámetros globales, gestionar planes de suscripción
│  No puede: ver datos de assessment de tenants individuales
│
└── TENANT ADMIN (por tenant)
    │  Puede: gestionar usuarios de su tenant, ver todos los dashboards,
    │         configurar el tenant, exportar informes, gestionar facturación
    │
    ├── OFICIAL DE PRIVACIDAD / DPO
    │   │  Puede: ejecutar diagnósticos, ver todos los resultados,
    │   │         asignar tareas de remediación, aprobar DPIAs,
    │   │         gestionar el RNBD, acceder a evidencias
    │   │  Restricción: no puede modificar configuraciones del tenant
    │
    ├── AUDITOR
    │   │  Puede: ver todos los dashboards y resultados (solo lectura),
    │   │         exportar informes de auditoría, ver evidencias
    │   │  No puede: modificar nada, asignar tareas
    │
    ├── EMPLEADO — ROL TI
    │   │  Puede: ver tareas de remediación asignadas a TI,
    │   │         cargar evidencias, marcar tareas completadas
    │   │  No puede: ver resultados de diagnóstico completos,
    │   │            acceder a datos de otros departamentos
    │
    └── EMPLEADO — ROL GENERAL
        │  Puede: ver tareas asignadas a su nombre,
        │         cargar evidencias de sus tareas
        │  No puede: ver diagnósticos, dashboards, ni otras tareas
```

#### 5.3.2 Matriz de Permisos Detallada

| Recurso | Superadmin | Tenant Admin | DPO | Auditor | Empleado TI | Empleado General |
|---|---|---|---|---|---|---|
| Dashboard Global | ✅ (todos los tenants) | ✅ (su tenant) | ✅ | ✅ (lectura) | ❌ | ❌ |
| GeoScore | ✅ | ✅ | ✅ | ✅ (lectura) | ❌ | ❌ |
| Diagnóstico — Ejecutar | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Diagnóstico — Ver resultados | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ |
| RNBD — Gestionar | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| DPIA — Crear/Editar | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| DPIA — Ver | ❌ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Tareas — Crear/Asignar | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Tareas — Ver asignadas | ❌ | ✅ | ✅ | ✅ | ✅ (solo suyas) | ✅ (solo suyas) |
| Evidencias — Cargar | ❌ | ✅ | ✅ | ❌ | ✅ (tareas suyas) | ✅ (tareas suyas) |
| Evidencias — Ver | ❌ | ✅ | ✅ | ✅ | ✅ (tareas suyas) | ✅ (tareas suyas) |
| Usuarios — Gestionar | ✅ (global) | ✅ (su tenant) | ❌ | ❌ | ❌ | ❌ |
| Tenants — Gestionar | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Informes — Exportar | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Configuración Tenant | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Logs de Auditoría | ✅ (todos) | ✅ (su tenant) | ✅ | ✅ (lectura) | ❌ | ❌ |

---

### 5.4 Modelo de Aislamiento Multitenant en Capa UI

#### 5.4.1 Aislamiento de Datos en Frontend

La capa de UI implementa múltiples capas de aislamiento:

**Capa 1 — Context Isolation:**
```typescript
// Contexto de tenant — todo componente del sistema consume este contexto
interface TenantContext {
  tenantId: string;         // UUID único del tenant
  tenantSlug: string;       // slug para URL (ej.: "empresa-xyz")
  tenantName: string;
  userRole: UserRole;
  userPermissions: Permission[];
  dataResidencyRegion: string; // "us-east-1" | "eu-west-1" | "sa-east-1"
}

// Ninguna llamada API se hace sin el tenantId en el header
// Interceptor global de Axios/fetch:
api.interceptors.request.use(config => {
  config.headers['X-Tenant-ID'] = tenantContext.tenantId;
  config.headers['X-Request-ID'] = generateUUID();
  return config;
});
```

**Capa 2 — URL Isolation:**
```
/app/{tenant-slug}/dashboard           → Dashboard del tenant
/app/{tenant-slug}/assessments         → Diagnósticos del tenant
/app/{tenant-slug}/remediation         → Tareas del tenant
/app/{tenant-slug}/admin               → Administración (solo Tenant Admin)

Regla: el {tenant-slug} es validado contra el JWT en cada request.
Un usuario no puede acceder a rutas de otro tenant aunque conozca el slug.
```

**Capa 3 — Component-Level Permission Guards:**
```typescript
// HOC de guardia de permisos
function PermissionGuard({
  requiredPermission,
  children,
  fallback = null
}: PermissionGuardProps) {
  const { userPermissions } = useTenantContext();
  
  if (!userPermissions.includes(requiredPermission)) {
    return fallback; // Renderiza null o un mensaje de "sin acceso"
  }
  
  return children;
}

// Uso:
<PermissionGuard requiredPermission="assessment.execute">
  <RunDiagnosticButton />
</PermissionGuard>
```

**Capa 4 — Data Masking en UI:**

Para el rol Auditor y Empleado, los datos personales reales de titulares (si aplica mostrarlos) se enmascaran en la UI:

```
Campo "Email del titular": johnd****@example.com
Campo "Nombre": J*** D***
Campo "Teléfono": +57 3** *** **89
```

---

*Fin del documento SDD_Arquitectura_Sistema_y_Negocio.md*
*Versión 1.0.0 — 2026-06-26*
