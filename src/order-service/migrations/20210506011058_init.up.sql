create table items
(
    id         serial  not null
        constraint items_pk
            primary key,
    order_id   integer not null,
    product_id integer not null,
    quantity   decimal not null default 0
        constraint items_quantity_nonnegative check (quantity >= 0)
);

create unique index items_id_uindex
    on items (id);

create unique index items_product_id_index
    on items (order_id, product_id);
