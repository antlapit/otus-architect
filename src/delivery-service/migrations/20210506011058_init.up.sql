create table delivery
(
    order_id   serial    not null
        constraint delivery_pk
            primary key,
    address    text      not null,
    date       timestamp not null,
    courier_id numeric
);

create table processed_orders
(
    order_id serial not null
        constraint processed_orders_pk
            primary key
);
