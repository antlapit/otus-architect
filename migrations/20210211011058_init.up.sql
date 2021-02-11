create table users
(
    id serial not null
        constraint users_pk
            primary key,
    username varchar(255) not null,
    first_name varchar(255) not null,
    last_name varchar(255) not null,
    email varchar(100) not null,
    phone varchar(100) not null
);

create unique index users_email_uindex
    on users (email);

create unique index users_id_uindex
    on users (id);

create unique index users_username_uindex
    on users (username);

create unique index users_phone_uindex
    on users (phone);



INSERT INTO users (username, first_name, last_name, email, phone) VALUES ('johndoe5892', 'John', 'Doe', 'bestjohn2@doe.com', '+710020030402');
