alter table products
    add column quantity decimal not null default 0
        constraint products_quantity_nonnegative check (quantity >= 0);
