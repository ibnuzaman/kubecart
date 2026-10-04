CREATE TABLE stock (
    sku      text PRIMARY KEY,
    -- Last line of defence against overselling, even if app code has a bug.
    quantity integer NOT NULL CHECK (quantity >= 0)
);
