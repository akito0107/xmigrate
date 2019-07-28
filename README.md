# xmigrate/pgmigrate

Schema first DB migration tool for PostgreSQL.

[![Build Status](https://dev.azure.com/akito01070362/xmigrate/_apis/build/status/akito0107.xmigrate?branchName=master)](https://dev.azure.com/akito01070362/xmigrate/_build/latest?definitionId=3&branchName=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/akito0107/xmigrate)](https://goreportcard.com/report/github.com/akito0107/xmigrate)
[![codecov](https://codecov.io/gh/akito0107/xmigrate/branch/master/graph/badge.svg)](https://codecov.io/gh/akito0107/xmigrate)

## Getting Started

### Prerequisites
- Go 1.12+

### Installing
```
$ go get -u github.com/akito0107/xmigrate/cmd/pgmigrate
```

### How To Use
1. Prepare Database.

```
# psql -U postgres
psql (9.6.10)

postgres=# create database xmigrate_tutorial;
CREATE DATABASE
```

2. Create `shcema` file.

```sql
CREATE TABLE ACCOUNT (
    id serial primary key,
    email varchar unique not null,
    name varchar,
    created_at timestamp with time zone default current_timestamp
);
```

3. Call sync command with `-f` option.

```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/xmigrate_tutorial
dry-run mode (with --apply flag will be exec below queries)
applying: CREATE TABLE ACCOUNT (id serial PRIMARY KEY, email character varying UNIQUE NOT NULL, name character varying, created_at timestamp with timezone DEFAULT current_timestamp)
```

4. Recall sync with `--apply` option.
```
$ pgmigrate -f schema.sql --apply postgres://[USER_NAME]:[PASSWORD]@localhost:5432/xmigrate_tutorial
applying: CREATE TABLE ACCOUNT (id serial PRIMARY KEY, email character varying UNIQUE NOT NULL, name character varying, created_at timestamp with time zone DEFAULT current_timestamp)
```

5. Check DB Stats
```
# psql -U postgres --dbname xmigrate_tutorial
psql (9.6.10)
Type "help" for help.

xmigrate_tutorial=# \d
               List of relations
 Schema |      Name      |   Type   |  Owner
--------+----------------+----------+----------
 public | account        | table    | postgres
 public | account_id_seq | sequence | postgres
(2 rows)

xmigrate_tutorial=# \d account
                                    Table "public.account"
   Column   |           Type           |                      Modifiers

------------+--------------------------+------------------------------------------------------
 id         | integer                  | not null default nextval('account_id_seq'::regclass)
 email      | character varying        | not null
 name       | character varying        |
 created_at | timestamp with time zone | default now()
Indexes:
    "account_pkey" PRIMARY KEY, btree (id)
    "account_email_key" UNIQUE CONSTRAINT, btree (email)
```

6. Modify Schema
```diff
CREATE TABLE ACCOUNT (
     id serial primary key,
     email varchar unique not null,
     name varchar,
+    address varchar not null,
     created_at timestamp with time zone default current_timestamp
 );
```

7. Preview & Apply
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE account ADD COLUMN address character varying NOT NULL

$ pgmigrate -f schema.sql --apply postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
applying: ALTER TABLE account ADD COLUMN address character varying NOT NULL
```

8. Check DB stats
```
xmigrate_tutorial=# \d account
                                    Table "public.account"
   Column   |           Type           |                      Modifiers
------------+--------------------------+------------------------------------------------------
 id         | integer                  | not null default nextval('account_id_seq'::regclass)
 email      | character varying        | not null
 name       | character varying        |
 created_at | timestamp with time zone | default now()
 address    | character varying        | not null
Indexes:
    "account_pkey" PRIMARY KEY, btree (id)
    "account_email_key" UNIQUE CONSTRAINT, btree (email)
```

## Supported Operation
### Create Table

given:
```diff
+CREATE TABLE ITEM (
+    ID serial primary key,
+    NAME varchar not null,
+    PRICE int not null
+);
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: CREATE TABLE ITEM (ID serial PRIMARY KEY, NAME character varying NOT NULL, PRICE int NOT NULL)
```

### Drop Table

given:
```diff
-CREATE TABLE ITEM (
-    ID serial primary key,
-    NAME varchar not null,
-    PRICE int not null
-);
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: DROP TABLE IF EXISTS item
```

### Add Column

given:
```diff
CREATE TABLE ACCOUNT (
     EMAIL varchar unique not null,
     NAME varchar,
     ADDRESS varchar not null,
+    ADDRESS2 varchar,
     CREATED_AT timestamp with time zone default current_timestamp
 );
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE account ADD COLUMN ADDRESS2 character varying
```

### Drop Column

given:
```diff
CREATE TABLE ACCOUNT (
     EMAIL varchar unique not null,
     NAME varchar,
     ADDRESS varchar not null,
-    ADDRESS2 varchar,
     CREATED_AT timestamp with time zone default current_timestamp
 );
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE account DROP COLUMN address2
```

### Modify Column
#### Change Data Type
    
given:
```diff
CREATE TABLE ACCOUNT (
     ID serial primary key,
     EMAIL varchar unique not null,
     NAME varchar,
-    ADDRESS varchar not null,
+    ADDRESS text not null,
     CREATED_AT timestamp with time zone default current_timestamp
 );
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE ACCOUNT ALTER COLUMN ADDRESS TYPE text
```

#### Add Column Constraint (NOT NULL)

given:
```diff
 CREATE TABLE ACCOUNT (
     ID serial primary key,
     EMAIL varchar unique not null,
-    NAME varchar,
+    NAME varchar not null,
     ADDRESS text not null,
     CREATED_AT timestamp with time zone default current_timestamp
 );
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE ACCOUNT ALTER COLUMN NAME SET NOT NULL
```

#### Remove Column Constraint (NOT NULL)

given:
```diff
 CREATE TABLE ACCOUNT (
     ID serial primary key,
     EMAIL varchar unique not null,
-    NAME varchar not null,
+    NAME varchar,
     ADDRESS text not null,
     CREATED_AT timestamp with time zone default current_timestamp
 );
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE ACCOUNT ALTER COLUMN NAME DROP NOT NULL
```

#### Add Table Constraint

given:
```diff
CREATE TABLE ACCOUNT (
     EMAIL varchar unique not null,
     NAME varchar not null,
     ADDRESS text not null,
     CREATED_AT timestamp with time zone default current_timestamp,
+    CONSTRAINT unique_name_address UNIQUE (NAME, ADDRESS)
 );
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE ACCOUNT ADD CONSTRAINT unique_name_address UNIQUE(NAME, ADDRESS)
```

CAUTION ONLY SUPPORTS *NAMED* CONSTRAINT. ex) the following case is not handled correctly.

```diff
CREATE TABLE ACCOUNT (
     EMAIL varchar unique not null,
     NAME varchar not null,
     ADDRESS text not null,
     CREATED_AT timestamp with time zone default current_timestamp,
+    UNIQUE (NAME, ADDRESS)
 );
```

#### Remove Table Constraint

given:
```diff
CREATE TABLE ACCOUNT (
     EMAIL varchar unique not null,
     NAME varchar not null,
     ADDRESS text not null,
     CREATED_AT timestamp with time zone default current_timestamp
-    CONSTRAINT unique_name_address UNIQUE (NAME, ADDRESS)
 );
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: ALTER TABLE ACCOUNT DROP CONSTRAINT unique_name_address
```

### Create Index

given:
```diff
+ CREATE INDEX name_idx ON account (name);
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
applying: CREATE INDEX cat_name_idx ON item (category_id, name)
```

### Drop Index

given:
```diff
- CREATE INDEX name_idx ON account (name);
```

then:
```
$ pgmigrate -f schema.sql postgres://[USER_NAME]:[PASSWORD]@localhost:5432/
dry-run mode (with --apply flag will be exec below queries)
DROP INDEX name_idx
```

### options
```
NAME:
   pgmigrate - postgres db migration utility

USAGE:
   pgmigrate [db url] [OPTIONS]

VERSION:
   0.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --schemapath value, -f value  schema sql path (default: "schema.sql")
   --apply, -a                   apply migration
   --help, -h                    show help
   --version, -v                 print the version
```

## License
This project is licensed under the Apache License 2.0 License - see the [LICENSE](LICENSE) file for details
