-- File: fixtures/seed-pgvector-test-data.sql
-- Purpose: Insert synthetic test data with deterministic embeddings for testing

BEGIN TRANSACTION;

-- 5 test chunks: 3 about neuroscience, 2 about programming
-- These embeddings are synthetic but consistent

INSERT INTO material_chunks 
  (id, material_id, topic_id, chunk_index, content, embedding) 
VALUES 
  ('00000000-0000-0000-0000-000000000001'::uuid,
   '550e8400-e29b-41d4-a716-446655440001'::uuid,
   '550e8400-e29b-41d4-a716-446655440002'::uuid,
   0,
   'Una sinapsis es la conexión entre neuronas. Los neurotransmisores se liberan desde la terminal pre-sináptica.',
   '[0.1,0.15,0.2,0.25,0.3,0.35,0.4,0.45,0.5,0.55,0.6,0.65,0.7,0.75,0.8,0.85,0.9,0.95,1.0,0.1,0.15,0.2,0.25,0.3]'::vector);

INSERT INTO material_chunks 
  (id, material_id, topic_id, chunk_index, content, embedding)
VALUES
  ('00000000-0000-0000-0000-000000000002'::uuid,
   '550e8400-e29b-41d4-a716-446655440001'::uuid,
   '550e8400-e29b-41d4-a716-446655440002'::uuid,
   1,
   'Los neurotransmisores como dopamina, serotonina y acetilcolina regulan el estado emocional y memoria.',
   '[0.12,0.17,0.22,0.27,0.32,0.37,0.42,0.47,0.52,0.57,0.62,0.67,0.72,0.77,0.82,0.87,0.92,0.97,0.99,0.12,0.17,0.22,0.27,0.32]'::vector);

INSERT INTO material_chunks 
  (id, material_id, topic_id, chunk_index, content, embedding)
VALUES
  ('00000000-0000-0000-0000-000000000003'::uuid,
   '550e8400-e29b-41d4-a716-446655440001'::uuid,
   '550e8400-e29b-41d4-a716-446655440002'::uuid,
   2,
   'La plasticidad neuronal permite al cerebro reorganizarse y formar nuevas conexiones sinápticas.',
   '[0.11,0.16,0.21,0.26,0.31,0.36,0.41,0.46,0.51,0.56,0.61,0.66,0.71,0.76,0.81,0.86,0.91,0.96,0.99,0.11,0.16,0.21,0.26,0.31]'::vector);

INSERT INTO material_chunks 
  (id, material_id, topic_id, chunk_index, content, embedding)
VALUES
  ('00000000-0000-0000-0000-000000000004'::uuid,
   '550e8400-e29b-41d4-a716-446655440001'::uuid,
   '550e8400-e29b-41d4-a716-446655440002'::uuid,
   3,
   'Python es un lenguaje de programación de alto nivel interpretado que soporta múltiples paradigmas.',
   '[0.95,0.9,0.85,0.8,0.75,0.7,0.65,0.6,0.55,0.5,0.45,0.4,0.35,0.3,0.25,0.2,0.15,0.1,0.05,0.95,0.9,0.85,0.8,0.75]'::vector);

INSERT INTO material_chunks 
  (id, material_id, topic_id, chunk_index, content, embedding)
VALUES
  ('00000000-0000-0000-0000-000000000005'::uuid,
   '550e8400-e29b-41d4-a716-446655440001'::uuid,
   '550e8400-e29b-41d4-a716-446655440002'::uuid,
   4,
   'JavaScript es el lenguaje para desarrollo web. Permite interactividad en navegadores y ejecución con Node.js.',
   '[0.93,0.88,0.83,0.78,0.73,0.68,0.63,0.58,0.53,0.48,0.43,0.38,0.33,0.28,0.23,0.18,0.13,0.08,0.03,0.93,0.88,0.83,0.78,0.73]'::vector);

COMMIT;

-- Validation: Check inserts
SELECT COUNT(*) as total_chunks FROM material_chunks;