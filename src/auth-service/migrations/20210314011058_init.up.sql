create table users
(
    id            serial       not null
        constraint users_pk
            primary key,
    username      varchar(255) not null,
    password      varchar(100) not null
);

create unique index users_username_uindex
    on users (username);
