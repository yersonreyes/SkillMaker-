-- Migration: 0001_init — DOWN
-- Description: Reverts the initial schema in reverse dependency order.
--   Indexes are dropped implicitly when tables are dropped.
--   pgcrypto extension is left in place (may be used by other tools/schemas).

DROP TABLE IF EXISTS refresh_token;
DROP TABLE IF EXISTS user_role;
DROP TABLE IF EXISTS role;
DROP TABLE IF EXISTS "user";
