CREATE TABLE tbl_orders
(
    id BIGSERIAL PRIMARY KEY,
    pair_id VARCHAR(25) NOT NULL,
    price DECIMAL(20, 10) NOT NULL,
    amount DECIMAL(20, 10) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    order_type int NOT NULL,

    account_id INTEGER REFERENCES tbl_accounts(id)
);



CREATE TABLE tbl_order_history_events (
    id SERIAL PRIMARY KEY,
    event VARCHAR(255) NOT NULL,
    status VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',

    order_id BIGINT REFERENCES tbl_orders(id)
);
