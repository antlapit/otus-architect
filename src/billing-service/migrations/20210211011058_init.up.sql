create table account
(
    id      serial  not null
        constraint account_pk
            primary key,
    user_id integer not null,
    money   integer not null default 0
        constraint money_nonnegative check (money >= 0)
);

create unique index account_id_uindex
    on account (id);

create unique index account_user_id_uindex
    on account (user_id);
