create table store_item
(
    product_id         serial  not null
        constraint store_item_pk
            primary key,
    available_quantity decimal not null default 0
        constraint store_item_available_quantity_nonnegative check (available_quantity >= 0)
);

create table processed_orders
(
    order_id serial not null
        constraint processed_orders_pk
            primary key
);
