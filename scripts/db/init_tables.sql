SET TIME ZONE 'UTC';


--- Таблица для основной информации о заказе
CREATE TABLE IF NOT EXISTS orders (
    order_uid VARCHAR(255) PRIMARY KEY,
    track_number VARCHAR(255) NOT NULL,
    entry VARCHAR(50) NOT NULL DEFAULT '',
    locale VARCHAR(10) NOT NULL DEFAULT '',
    internal_signature VARCHAR(255) NOT NULL DEFAULT '',
    customer_id VARCHAR(255) NOT NULL DEFAULT '',
    delivery_service VARCHAR(255) NOT NULL DEFAULT '',
    shardkey VARCHAR(10) NOT NULL DEFAULT '',
    sm_id INT NOT NULL DEFAULT 0,
    date_created TIMESTAMPTZ NOT NULL,
    oof_shard VARCHAR(10) NOT NULL DEFAULT ''
);


--- Таблица для информации о доставке (связь один-к-одному с orders)
CREATE TABLE IF NOT EXISTS delivery (
    -- Связываем с заказом через внешний ключ, который также является первичным ключом
    order_uid VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT '',
    phone VARCHAR(50) NOT NULL DEFAULT '',
    zip VARCHAR(50) NOT NULL DEFAULT '',
    city VARCHAR(255) NOT NULL DEFAULT '',
    address VARCHAR(255) NOT NULL DEFAULT '',
    region VARCHAR(255) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL DEFAULT ''
);


--- Таблица для информации об оплате (связь один-к-одному с orders)
CREATE TABLE IF NOT EXISTS payment (
    order_uid VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    request_id VARCHAR(255) NOT NULL DEFAULT '',
    currency VARCHAR(10) NOT NULL DEFAULT '',
    provider VARCHAR(50) NOT NULL DEFAULT '',
    amount INT NOT NULL DEFAULT 0,
    payment_dt TIMESTAMPTZ,
    bank VARCHAR(255) NOT NULL DEFAULT '',
    delivery_cost INT NOT NULL DEFAULT 0,
    goods_total INT NOT NULL DEFAULT 0,
    custom_fee INT NOT NULL DEFAULT 0
);


--- Таблица для товаров в заказе (связь один-ко-многим с orders)
CREATE TABLE IF NOT EXISTS items (
    rid VARCHAR(255) PRIMARY KEY,
    order_uid VARCHAR(255) NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id INT NOT NULL DEFAULT 0,
    track_number VARCHAR(255) NOT NULL DEFAULT '',
    price INT NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL DEFAULT '',
    sale INT NOT NULL DEFAULT 0,
    size VARCHAR(50) NOT NULL DEFAULT '',
    total_price INT NOT NULL DEFAULT 0,
    nm_id INT NOT NULL DEFAULT 0,
    brand VARCHAR(255) NOT NULL DEFAULT '',
    status INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);