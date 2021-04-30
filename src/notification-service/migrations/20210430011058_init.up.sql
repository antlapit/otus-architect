create table notification
(
    id      serial  not null
        constraint notification_pk
            primary key,
    user_id integer not null,
    order_id integer not null,
    event_id varchar(100) not null,
    event_type varchar(100) not null,
    event_data text not null
);

create unique index notification_id_uindex
    on notification (id);

create index notification_user_id_index
    on notification (user_id);

create index notification_event_uindex
    on notification (order_id, event_type);
