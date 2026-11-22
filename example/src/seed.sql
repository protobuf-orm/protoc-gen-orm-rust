CREATE TABLE people (
    id INTEGER PRIMARY KEY,
    tenant INTEGER NOT NULL,
    alias TEXT NOT NULL,
    UNIQUE (tenant, alias)
);
INSERT INTO people (id, tenant, alias) VALUES (17, 1, 'sam'), (42, 2, 'sam');
