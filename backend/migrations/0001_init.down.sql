-- 0001_init.down.sql
-- Reverts 0001_init.up.sql. Drops in dependency order.

BEGIN;

DROP TRIGGER IF EXISTS trg_documents_search_vector ON documents;
DROP FUNCTION IF EXISTS documents_search_vector_update();

DROP TRIGGER IF EXISTS trg_documents_set_updated_at ON documents;
DROP FUNCTION IF EXISTS set_updated_at();

DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS activity_logs;
DROP TABLE IF EXISTS document_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS documents;

DROP TYPE IF EXISTS activity_action;
DROP TYPE IF EXISTS document_section;

COMMIT;
