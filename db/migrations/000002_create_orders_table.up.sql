CREATE TABLE IF NOT EXISTS tbl_orders
(
    id BIGSERIAL PRIMARY KEY,
    pair_id VARCHAR(25) NOT NULL,
    price DECIMAL(20, 10) NOT NULL,
    amount DECIMAL(20, 10) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    order_type INTEGER NOT NULL,

    account_id INTEGER REFERENCES tbl_accounts(id)
);
