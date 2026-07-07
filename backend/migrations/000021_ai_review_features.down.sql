-- PostgreSQL does not support removing values from an enum type, so this down
-- migration is intentionally a no-op. Rolling back leaves the (unused) enum
-- values 'code_review' and 'lld_review' in place, which is harmless. Fix forward
-- if the values must truly be removed (recreate the type).
SELECT 1;
