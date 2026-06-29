-- 000018_design_problems_extra.down.sql

BEGIN;

DELETE FROM design_problems WHERE slug IN (
    'youtube', 'uber', 'google-docs', 'payment-gateway', 'slack', 'rate-limiter'
);

COMMIT;
