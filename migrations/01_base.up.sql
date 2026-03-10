begin;

create table grp (
  id uuid primary key default gen_random_uuid(),
  name varchar(64) unique
);

create table person (
  id uuid primary key default gen_random_uuid(),
  username varchar(64) not null unique,
  grp uuid not null references grp (id)
);

create table suggestion (
  id uuid primary key default gen_random_uuid(),
  suggester uuid not null references person (id) on delete cascade,
  regarding uuid not null references person (id) on delete cascade,
  suggestion text,
  motivation text,
  -- A person cannot make two suggestions regarding a single person.
  unique (suggester, regarding),
  -- A person cannot make a suggestion regarding themselves.
  check (suggester != regarding),
  -- Support composite foreign key.
  unique (id, regarding)
);

create table vote (
  id uuid primary key default gen_random_uuid(),
  caster uuid not null references person (id) on delete cascade,
  target_suggestion uuid not null references suggestion (id) on delete cascade,
  regarding uuid not null references person (id) on delete cascade,
  time timestamptz not null default now(),
  -- A caster cannot vote twice for the same suggestion.
  unique (caster, target_suggestion),
  -- A caster cannot vote for a suggestion regarding themselves.
  check (caster != regarding),
  -- A caster cannot vote for multiple suggestions about the same person.
  unique (caster, regarding),
  -- Enforce consistency on the regarding column.
  foreign key (target_suggestion, regarding) references suggestion (id, regarding)
);

commit;
