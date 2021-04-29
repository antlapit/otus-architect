create table bill
(
    id serial not null
        constraint bill_pk
            primary key,
    account_id integer not null,
    order_id integer not null,
    status varchar(100) not null,
    total decimal not null
);
