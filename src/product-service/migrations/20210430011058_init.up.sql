create table products
(
    id      serial  not null
        constraint products_pk
            primary key,
    name varchar(1000) not null,
    description TEXT not null,
    archived boolean  not null default false
);

create unique index products_id_uindex
    on products (id);
