begin;

alter table person
drop column password_last_changed;

commit;
