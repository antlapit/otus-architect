alter table orders
    add column warehouse_confirmed bool not null default false,
    add column delivery_confirmed bool not null default false;
