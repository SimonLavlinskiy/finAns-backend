CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    login         VARCHAR(64) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO users (login, password_hash) VALUES
    ('anna', '$2a$10$Df2iYXvD.PpV2KdDnOp21OXOWfMihvYapEyj.4vJBm8qeVXLeS4m2'),
    ('simon', '$2a$10$/Hz6zyIpTHRJHmY56KbxNOHO535vD0jHJzpDCNBESeO9/GhQ.Yk8y');
