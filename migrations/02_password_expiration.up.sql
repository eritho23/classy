begin;

alter table person
add column password_last_changed timestamptz default null;

commit;
