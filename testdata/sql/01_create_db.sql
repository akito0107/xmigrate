create role xmigrate_user with login password 'passw0rd';
create database xmigrate_test;
grant all privileges on database xmigrate_test to xmigrate_user;

