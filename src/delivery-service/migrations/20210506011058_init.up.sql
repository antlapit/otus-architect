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

create table courier
(
    courier_id  serial  not null
        constraint courier_pk
            primary key,
    max_per_day numeric not null default 1
)
