alter table products
    add column min_price decimal not null default 0
        constraint products_min_price_nonnegative check (min_price >= 0),
    add column max_price decimal not null default 0
        constraint products_max_price_nonnegative check (max_price >= 0),
    drop column price;
