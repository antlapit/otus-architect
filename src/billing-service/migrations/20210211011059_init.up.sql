create table bill
(
    id         serial       not null
        constraint bill_pk
            primary key,
    account_id integer      not null,
    order_id   integer      not null,
    status     varchar(100) not null,
    total      decimal      not null default 0
        constraint bill_total_nonnegative check (total >= 0)
);
