begin;

alter table challenges
drop column batch_number;

commit;
