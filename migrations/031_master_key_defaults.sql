-- 031_master_key_defaults.sql
-- Ensure records inserted after the master/detail migration receive business keys.

DO $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT table_name, key_prefix
        FROM data_table_catalog
        ORDER BY table_name
    LOOP
        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = rec.table_name
              AND column_name = 'master_key'
        ) THEN
            EXECUTE FORMAT(
                'ALTER TABLE %I ALTER COLUMN master_key SET DEFAULT next_business_key(%L, %L)',
                rec.table_name,
                rec.table_name,
                rec.key_prefix
            );
            EXECUTE FORMAT(
                'UPDATE %I SET master_key = next_business_key(%L, %L) WHERE master_key IS NULL',
                rec.table_name,
                rec.table_name,
                rec.key_prefix
            );
            EXECUTE FORMAT('ALTER TABLE %I ALTER COLUMN master_key SET NOT NULL', rec.table_name);
        END IF;
    END LOOP;
END;
$$;
