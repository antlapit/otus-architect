alter table orders
    add column date timestamp default now();

update orders
set date = now();

alter table orders
    alter column date set not null;
