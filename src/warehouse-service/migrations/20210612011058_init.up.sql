create table event_message
(
    id         varchar(100) not null
        constraint event_message_pk
            primary key,
    data       text         not null,
    status     varchar(100) not null,
    created_at timestamp    not null
);
