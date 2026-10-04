-- Demo data. cherry starts empty so FailedPrecondition is easy to try.
INSERT INTO stock (sku, quantity) VALUES
    ('apple', 10),
    ('banana', 5),
    ('cherry', 0);
