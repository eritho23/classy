begin;

alter table challenges
add column batch_number integer not null default 1;

commit;
