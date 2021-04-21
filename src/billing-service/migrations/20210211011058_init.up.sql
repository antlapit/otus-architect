create table account
(
    id serial not null
        constraint user_profile_pk
            primary key,
    user_id integer not null,
    money integer not null default 0
);

create unique index account_id_uindex
    on account (id);

create unique index account_user_id_uindex
    on account (user_id);
