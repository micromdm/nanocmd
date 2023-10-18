CREATE TABLE steps (
    id   BIGINT  NOT NULL AUTO_INCREMENT PRIMARY KEY,

    workflow_name VARCHAR(255) NOT NULL,
    instance_id   VARCHAR(255) NOT NULL,
    step_name     VARCHAR(255) NULL,

    context MEDIUMTEXT NULL,

    not_until      TIMESTAMP NULL,
    timeout        TIMESTAMP NULL,

    process_id     CHAR(45) NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE id_commands (
    enrollment_id VARCHAR(255) NOT NULL,
    command_uuid  VARCHAR(127) NOT NULL,

    step_id BIGINT NOT NULL,

    request_type VARCHAR(63)  NOT NULL,
    completed    BOOLEAN      NOT NULL DEFAULT 0,
    result       MEDIUMTEXT   NULL,

    last_push TIMESTAMP NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (step_id)
        REFERENCES steps (id),

    PRIMARY KEY (enrollment_id, command_uuid)
);

CREATE TABLE step_commands (
    command_uuid VARCHAR(127) NOT NULL,
    step_id      BIGINT       NOT NULL,

    command      MEDIUMTEXT   NOT NULL,
    request_type VARCHAR(63)  NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (step_id)
        REFERENCES steps (id),

    PRIMARY KEY (command_uuid, step_id)
);

CREATE TABLE wf_events (
    event_name VARCHAR(255) NOT NULL,

    context       MEDIUMTEXT   NULL,
    workflow_name VARCHAR(255) NOT NULL,
    event_type    VARCHAR(63)  NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (event_name)
);