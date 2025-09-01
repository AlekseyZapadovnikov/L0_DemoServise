-- Создаём роль, только если она ещё не существует.
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'order_user') THEN
        CREATE ROLE order_user WITH LOGIN PASSWORD '111';
    END IF;
END $$;