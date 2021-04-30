create table order
(
    id      serial  not null
        constraint order_pk
            primary key,
    user_id integer not null,
    status varchar(100) not null,
    amount decimal  not null default 0
        constraint order_amount_nonnegative check (amount >= 0)
);

create unique index order_id_uindex
    on order (id);

create index order_user_id_index
    on order (user_id);
