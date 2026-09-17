-- 0001_init.up.sql
-- Initial schema for self-docs (PRD section 6.1).
-- Entities: documents, tags, document_tags, activity_logs, settings.
-- See docs/data-model.md for the frozen data model.

BEGIN;

-- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------------------
-- Enumerations (docs/data-model.md section 2)
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_section') THEN
        CREATE TYPE document_section AS ENUM (
            'workflow',
            'project_note',
            'cheat_sheet',
            'private'
        );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'activity_action') THEN
        CREATE TYPE activity_action AS ENUM (
            'created',
            'updated',
            'deleted',
            'imported',
            'unlocked'
        );
    END IF;
END
$$;

-- ---------------------------------------------------------------------------
-- documents (docs/data-model.md section 3.1)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS documents (
    id            uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
    title         text             NOT NULL,
    slug          text             NOT NULL,
    content       text             NOT NULL DEFAULT '',
    excerpt       text,
    section       document_section NOT NULL,
    parent_id     uuid             REFERENCES documents (id) ON DELETE CASCADE,
    position      integer          NOT NULL DEFAULT 0,
    is_private    boolean          NOT NULL DEFAULT false,
    search_vector tsvector,
    created_at    timestamptz      NOT NULL DEFAULT now(),
    updated_at    timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT documents_title_len_chk CHECK (char_length(title) BETWEEN 1 AND 300),
    CONSTRAINT documents_slug_not_empty_chk CHECK (char_length(slug) > 0),
    CONSTRAINT documents_parent_only_project_note_chk
        CHECK (section = 'project_note' OR parent_id IS NULL),
    CONSTRAINT documents_no_self_parent_chk CHECK (parent_id IS DISTINCT FROM id),
    CONSTRAINT documents_is_private_derived_chk
        CHECK (is_private = (section = 'private'))
);

-- Root-level slugs are unique within a section; nested slugs are unique among
-- siblings. NULL parent_id values are distinct in a plain unique index, so a
-- partial index is used for the root case (docs/data-model.md section 3.1).
CREATE UNIQUE INDEX IF NOT EXISTS uq_documents_section_slug_root
    ON documents (section, slug)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_documents_parent_slug
    ON documents (parent_id, slug)
    WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_documents_section
    ON documents (section);

CREATE INDEX IF NOT EXISTS idx_documents_parent_id
    ON documents (parent_id);

CREATE INDEX IF NOT EXISTS idx_documents_updated_at
    ON documents (updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_documents_is_private
    ON documents (is_private);

CREATE INDEX IF NOT EXISTS idx_documents_search_vector
    ON documents USING gin (search_vector);

-- ---------------------------------------------------------------------------
-- tags (docs/data-model.md section 3.2)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS tags (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text        NOT NULL,
    normalized text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tags_name_len_chk CHECK (char_length(name) BETWEEN 1 AND 64),
    CONSTRAINT tags_normalized_not_empty_chk CHECK (char_length(normalized) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_normalized
    ON tags (normalized);

CREATE INDEX IF NOT EXISTS idx_tags_name
    ON tags (name);

-- ---------------------------------------------------------------------------
-- document_tags (docs/data-model.md section 3.3)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS document_tags (
    document_id uuid NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    tag_id      uuid NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (document_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_document_tags_tag_id
    ON document_tags (tag_id);

-- ---------------------------------------------------------------------------
-- activity_logs (docs/data-model.md section 3.4)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS activity_logs (
    id             uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
    action         activity_action  NOT NULL,
    document_id    uuid             REFERENCES documents (id) ON DELETE SET NULL,
    document_title text,
    section        document_section,
    actor          text             NOT NULL DEFAULT 'user',
    created_at     timestamptz      NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at
    ON activity_logs (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_activity_logs_document_id
    ON activity_logs (document_id);

-- ---------------------------------------------------------------------------
-- settings (docs/data-model.md section 3.5)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS settings (
    key        text        PRIMARY KEY,
    value      text        NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- updated_at maintenance
--
-- Prefer a trigger so every UPDATE bumps updated_at regardless of call site.
-- The repository layer must not set updated_at manually.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_documents_set_updated_at ON documents;
CREATE TRIGGER trg_documents_set_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Full-text search maintenance (docs/data-model.md section 5)
--
-- title weighted A, content weighted B, using the 'simple' dictionary so code
-- identifiers and mixed-language content match literally. Tags are joined at
-- query time and are not part of this vector.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION documents_search_vector_update()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('simple', coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(NEW.content, '')), 'B');
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_documents_search_vector ON documents;
CREATE TRIGGER trg_documents_search_vector
    BEFORE INSERT OR UPDATE OF title, content ON documents
    FOR EACH ROW
    EXECUTE FUNCTION documents_search_vector_update();

INSERT INTO settings (key, value)
VALUES ('master_password_hash', '')
ON CONFLICT (key) DO NOTHING;

COMMIT;
