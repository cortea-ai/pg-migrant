CREATE TABLE "public"."users" (
	"id" uuid NOT NULL DEFAULT uuid_generate_v4(),
	"contact_id" uuid NOT NULL,
	"created_at" timestamp with time zone NOT NULL DEFAULT now(),
	"updated_at" timestamp with time zone NOT NULL DEFAULT now()
);
-- END STATEMENT --

ALTER TABLE "public"."users" ADD CONSTRAINT "users_contact_id_fkey" FOREIGN KEY (contact_id) REFERENCES contacts(id) NOT VALID;
-- END STATEMENT --

ALTER TABLE "public"."users" VALIDATE CONSTRAINT "users_contact_id_fkey";
-- END STATEMENT --

CREATE UNIQUE INDEX CONCURRENTLY users_pkey ON public.users USING btree (id);
-- [NOTE]: This statement cannot run in a transaction with other statements, place in a separate migration.
-- END STATEMENT --

ALTER TABLE "public"."users" ADD CONSTRAINT "users_pkey" PRIMARY KEY USING INDEX "users_pkey";
