ALTER TABLE wf_status
    DROP COLUMN last_created_at,
    ADD COLUMN last_created_unix BIGINT NOT NULL DEFAULT (UNIX_TIMESTAMP());
