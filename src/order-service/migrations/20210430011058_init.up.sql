create table orders
(
    id      serial  not null
        constraint orders_pk
            primary key,
    user_id integer not null,
    status varchar(100) not null,
    total decimal  not null default 0
        constraint order_total_nonnegative check (total >= 0)
);

create unique index order_id_uindex
    on orders (id);

create index order_user_id_index
    on orders (user_id);
