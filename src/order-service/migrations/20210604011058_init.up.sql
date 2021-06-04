alter table items
    add column base_price decimal not null default 0
        constraint items_base_price_nonnegative check (base_price >= 0),
    add column calc_price decimal not null default 0
        constraint items_calc_price_nonnegative check (calc_price >= 0),
    add column total decimal not null default 0
        constraint items_total_nonnegative check (total >= 0)
