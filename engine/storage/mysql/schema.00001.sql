ALTER TABLE wf_events ADD COLUMN event_context MEDIUMTEXT NULL;
CREATE TABLE wf_status (
    enrollment_id VARCHAR(255) NOT NULL,
    workflow_name VARCHAR(255) NOT NULL,

    last_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX (enrollment_id),
    INDEX (workflow_name),

    PRIMARY KEY (enrollment_id, workflow_name)
);
