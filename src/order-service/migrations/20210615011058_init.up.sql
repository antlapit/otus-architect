alter table orders
    add column changes text default '';

update order set changes = '';

