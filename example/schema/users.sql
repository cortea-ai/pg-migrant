CREATE TABLE users (
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    contact_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    CONSTRAINT users_contact_id_fkey FOREIGN KEY (contact_id) REFERENCES contacts(id)
);
