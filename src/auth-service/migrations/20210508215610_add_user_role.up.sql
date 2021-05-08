ALTER TABLE users ADD COLUMN role varchar(100);

UPDATE users SET role = 'USER';

ALTER TABLE users ALTER COLUMN role SET NOT NULL;
