create table products
(
    id      serial  not null
        constraint products_pk
            primary key,
    name varchar(1000) not null,
    description TEXT not null,
    archived boolean  not null default false,
    category_id numeric[],
    price decimal  not null default 0
        constraint product_price_nonnegative check (price >= 0)
);

create unique index products_id_uindex
    on products (id);
