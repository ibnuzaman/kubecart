-- No foreign key to inventory's stock table: it lives in another database
-- owned by another service. order-service only knows a SKU string.
CREATE TABLE orders (
    id         text PRIMARY KEY,
    sku        text NOT NULL,
    quantity   integer NOT NULL CHECK (quantity > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);
