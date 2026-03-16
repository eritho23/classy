-- name: CreateGroup :one
insert into
  grp (name)
values
  ($1)
on conflict do nothing
returning
  id,
  name;

-- name: GetGroupByName :one
select
  id,
  name
from
  grp
where
  name = $1
limit
  1;

-- name: CreatePerson :one
insert into
  person (username, grp, password_hash)
values
  ($1, $2, $3)
returning
  id,
  username,
  grp;

-- name: GetPersonByUsername :one
select
  id,
  username,
  grp
from
  person
where
  username = $1
limit
  1;

-- name: GetPersonPasswordHashByUsername :one
select
  id,
  password_hash,
  username
from
  person
where
  username = $1
limit
  1;

-- name: DeletePersonByUsername :exec
delete from person
where
  username = $1;

-- name: CreateSuggestion :one
insert into
  suggestion (suggester, regarding, suggestion, motivation)
values
  ($1, $2, $3, $4)
returning
  id,
  suggester,
  regarding,
  suggestion,
  motivation;

-- name: GetSuggestionById :one
select
  id,
  suggester,
  regarding,
  suggestion,
  motivation
from
  suggestion
where
  id = $1;

-- name: UpdateSuggestion :exec
update suggestion
set
  suggestion = $1,
  motivation = $2
where
  id = $3;

-- name: DeleteSuggestion :exec
delete from suggestion
where
  id = $1;

-- These two will be wrapped in a transaction block later.
-- name: CreateVote :one
insert into
  vote (caster, target_suggestion, regarding)
select
  $1,
  $2,
  regarding
from
  suggestion
where
  id = $2
returning
  id,
  caster,
  target_suggestion,
  regarding,
  time;

-- name: DeleteVoteByCasterAndSuggestion :exec
delete from vote
where
  caster = $1
  and target_suggestion = $2;

-- name: CreateSession :one
insert into
  session (value, expires_at, created_at, person)
values
  ($1, $2, $3, $4)
returning
  created_at,
  expires_at,
  id;

-- name: GetSessionByValue :one
select
  session.id,
  value,
  created_at,
  expires_at,
  session.person,
  person.username
from
  session
  inner join person on session.person = person.id
where
  value = $1
limit
  1;

-- name: DeleteSessionById :exec
delete from session
where
  id = $1;
