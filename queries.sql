-- name: CreateGroup :one
insert into
  grp (name)
values
  ($1)
on conflict do nothing
returning
  uid,
  name;

-- name: GetGroupByUid :one
select
  uid,
  name
from
  grp
where
  uid = $1;

-- name: GetGroupByUsername :one
select
  person.username as username,
  person.uid as person_uid,
  grp.uid as group_uid,
  grp.name as group_name
from
  grp
  inner join person on person.grp = grp.uid
where
  person.username = $1
limit
  1;

-- name: CreatePerson :one
insert into
  person (username, grp, password_hash)
values
  ($1, $2, $3)
returning
  uid,
  username,
  grp;

-- name: GetPersonByUsername :one
select
  uid,
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
  uid,
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

-- name: GetStudentsAndSuggestionCountsByGrp :many
select
  username,
  uid,
  (
    select
      count(*)
    from
      suggestion
    where
      regarding = uid
  ) as number_of_suggestions
from
  person
where
  grp = $1
order by
  username asc;

-- name: CreateSuggestion :one
insert into
  suggestion (suggester, regarding, suggestion, motivation)
values
  ($1, $2, $3, $4)
returning
  uid,
  suggester,
  regarding,
  suggestion,
  motivation;

-- name: GetSuggestionByUid :one
select
  uid,
  suggester,
  regarding,
  suggestion,
  motivation
from
  suggestion
where
  uid = $1;

-- name: UpdateSuggestion :exec
update suggestion
set
  suggestion = $1,
  motivation = $2
where
  uid = $3;

-- name: DeleteSuggestion :exec
delete from suggestion
where
  uid = $1;

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
  uid = $2
returning
  uid,
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
  uid;

-- name: GetSessionByValue :one
select
  session.uid,
  value,
  created_at,
  expires_at,
  session.person,
  person.username
from
  session
  inner join person on session.person = person.uid
where
  value = $1
limit
  1;

-- name: DeleteSessionByUid :exec
delete from session
where
  uid = $1;

-- name: GetGroupAndPersonPartOfGroupByGroupuid :one
select
  grp.uid,
  grp.name,
  exists (
    select
      1
    from
      person p
    where
      p.grp = grp.uid
      and p.uid = @person_uid
  ) as person_part_of_group
from
  grp
where
  grp.uid = @group_uid
limit
  1;
