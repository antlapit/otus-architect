create table user_profile
(
    id serial not null
        constraint user_profile_pk
            primary key,
    first_name varchar(255),
    last_name varchar(255),
    email varchar(100),
    phone varchar(100)
);

create unique index user_profile_id_uindex
    on user_profile (id);

create unique index user_profile_email_uindex
    on user_profile (email);

create unique index user_profile_phone_uindex
    on user_profile (phone);
