create table account (
    account_id serial primary key,
    name varchar(255),
    email varchar(255) unique not null,
    age smallint not null,
    registered_at timestamp with time zone default current_timestamp
);

CREATE INDEX name_idx ON account (name);

create table category (
    category_id serial primary key,
    name varchar(255) not null
);

create table item (
    item_id serial primary key,
    price int not null,
    name varchar not null,
    category_id int references category(category_id),
    created_at timestamp with time zone default current_timestamp,
    CONSTRAINT unique_price_and_name UNIQUE (price, name)
);

create index cat_name_idx on item (category_id, name);

create table subitem(
    subitem_id serial primary key,
    price int not null,
    name varchar not null,
    CONSTRAINT fkey_item_id_name FOREIGN KEY (price, name) REFERENCES item(price, name)
);

create table subcategory(
    name varchar not null,
    category_id  int not null,
    CONSTRAINT pkey_name_category_id PRIMARY KEY (name, category_id)
);

