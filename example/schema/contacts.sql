CREATE TABLE contacts (
    id uuid NOT NULL DEFAULT uuid_generate_v4(),
    full_name character varying(255) NULL,
    email character varying(255) NOT NULL,
    phone character varying(255) NULL,
    address character varying(255) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id),
    CONSTRAINT contacts_email_key UNIQUE (email)
);
