alter table products
    drop column min_price,
    drop column max_price,
    add column price decimal not null default 0
        constraint products_price_nonnegative check (price >= 0);
