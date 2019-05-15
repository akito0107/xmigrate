

create table account (
    account_id serial primary key,
    name varchar(255) not null,
    email varchar(255) unique not null,
    age smallint not null,
    registered_at timestamp with time zone default current_timestamp
);

create table category (
    category_id serial primary key,
    name varchar(255) not null
);

create table item (
    item_id serial primary key,
    price int not null,
    name varchar(255) not null,
    category_id int references category(category_id),
    created_at timestamp with time zone default current_timestamp
);

