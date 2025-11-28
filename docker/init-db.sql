
-- Create test database
CREATE DATABASE testdb;

-- Connect to testdb
\c testdb

-- Enable pg_stat_statements in testdb as well
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create test table: users
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- Create index on email
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active ON users(is_active);

-- Create test table: orders
CREATE TABLE orders (
    order_id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(user_id),
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_amount NUMERIC(10, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    shipped_at TIMESTAMP
);

-- Create indexes on orders
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_date ON orders(order_date);

-- Insert test data into users
INSERT INTO users (username, email, created_at, last_login, is_active) VALUES
    ('alice', 'alice@example.com', NOW() - INTERVAL '30 days', NOW() - INTERVAL '1 day', true),
    ('bob', 'bob@example.com', NOW() - INTERVAL '25 days', NOW() - INTERVAL '2 days', true),
    ('charlie', 'charlie@example.com', NOW() - INTERVAL '20 days', NOW() - INTERVAL '5 days', true),
    ('david', 'david@example.com', NOW() - INTERVAL '15 days', NULL, false),
    ('eve', 'eve@example.com', NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 hour', true),
    ('frank', 'frank@example.com', NOW() - INTERVAL '5 days', NOW() - INTERVAL '3 hours', true),
    ('grace', 'grace@example.com', NOW() - INTERVAL '3 days', NOW() - INTERVAL '30 minutes', true),
    ('henry', 'henry@example.com', NOW() - INTERVAL '1 day', NULL, false);

-- Insert test data into orders
INSERT INTO orders (user_id, order_date, total_amount, status, shipped_at) VALUES
    (1, NOW() - INTERVAL '29 days', 99.99, 'delivered', NOW() - INTERVAL '28 days'),
    (1, NOW() - INTERVAL '15 days', 149.50, 'delivered', NOW() - INTERVAL '14 days'),
    (1, NOW() - INTERVAL '2 days', 75.00, 'shipped', NOW() - INTERVAL '1 day'),
    (2, NOW() - INTERVAL '24 days', 199.99, 'delivered', NOW() - INTERVAL '23 days'),
    (2, NOW() - INTERVAL '10 days', 49.99, 'delivered', NOW() - INTERVAL '9 days'),
    (3, NOW() - INTERVAL '18 days', 299.00, 'delivered', NOW() - INTERVAL '17 days'),
    (3, NOW() - INTERVAL '5 days', 125.50, 'processing', NULL),
    (5, NOW() - INTERVAL '8 days', 89.99, 'delivered', NOW() - INTERVAL '7 days'),
    (5, NOW() - INTERVAL '3 days', 179.99, 'shipped', NOW() - INTERVAL '2 days'),
    (5, NOW() - INTERVAL '1 hour', 45.00, 'pending', NULL),
    (6, NOW() - INTERVAL '4 days', 250.00, 'processing', NULL),
    (7, NOW() - INTERVAL '2 days', 99.99, 'shipped', NOW() - INTERVAL '1 day'),
    (7, NOW() - INTERVAL '6 hours', 150.00, 'pending', NULL);

-- Create a view for testing
CREATE VIEW user_order_summary AS
SELECT
    u.user_id,
    u.username,
    u.email,
    COUNT(o.order_id) as total_orders,
    COALESCE(SUM(o.total_amount), 0) as total_spent,
    MAX(o.order_date) as last_order_date
FROM users u
LEFT JOIN orders o ON u.user_id = o.user_id
GROUP BY u.user_id, u.username, u.email;

-- Generate some query activity for pg_stat_statements
SELECT COUNT(*) FROM users WHERE is_active = true;
SELECT COUNT(*) FROM orders WHERE status = 'pending';
SELECT * FROM user_order_summary ORDER BY total_spent DESC LIMIT 5;

-- Create a function for testing
CREATE OR REPLACE FUNCTION get_user_stats()
RETURNS TABLE(
    total_users BIGINT,
    active_users BIGINT,
    inactive_users BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT as total_users,
        COUNT(*) FILTER (WHERE is_active = true)::BIGINT as active_users,
        COUNT(*) FILTER (WHERE is_active = false)::BIGINT as inactive_users
    FROM users;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;

-- Print success message
DO $$
BEGIN
    RAISE NOTICE 'Test database initialized successfully!';
    RAISE NOTICE 'Created tables: users, orders';
    RAISE NOTICE 'Created view: user_order_summary';
    RAISE NOTICE 'Created function: get_user_stats()';
    RAISE NOTICE 'Inserted % users and % orders',
        (SELECT COUNT(*) FROM users),
        (SELECT COUNT(*) FROM orders);
END $$;
