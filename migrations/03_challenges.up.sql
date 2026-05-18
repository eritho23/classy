begin;

create table challenges (
  uid uuid primary key default gen_random_uuid(),
  description text,
  assigned_number int not null,
  points int not null,
  extra_points_available boolean not null default false,
  extra_points_received int not null default 0,
  completed_by uuid references person (uid)
);

commit;
