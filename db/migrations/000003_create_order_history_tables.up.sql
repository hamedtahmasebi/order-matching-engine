CREATE TABLE IF NOT EXISTS tbl_order_history_events (
    id SERIAL PRIMARY KEY,
    event VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',

    order_id BIGINT REFERENCES tbl_orders(id)
);
