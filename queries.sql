-- name: CreateGroup :one
insert into group (name) values ($1) returning id, name;

-- name: GetGroupByName :one
select id, name from group where name = $1 limit 1;

-- name: CreatePerson :one
insert into person (username, group) values ($1, $2) returning id, username, group;

-- name: GetPersonByUsername :one
select id, username, group from person where username = $1 limit 1;

-- name: DeletePersonByUsername :exec
delete from person where username = $1 limit 1;

-- name: CreateSuggestion :one
insert into suggestion (suggester, regarding, suggestion, motivation) returning
id,
suggester,
regarding,
suggestion,
motivation;

-- name: GetSuggestionById :one
select id,
suggester,
regarding,
suggestion,
motivation from suggestion where id = $1;

-- name: UpdateSuggestion :exec
update suggestion set suggestion = $1, motivation = $2 where id = $3;

-- name: DeleteSuggestion :exec
delete from suggestion where id = $1;

-- These two will be wrapped in a transaction block later.

-- name: CreateVote :one
insert into vote (caster, target_suggestion, regarding)
select
@caster::uuid,
@target_suggestion::uuid,
regarding
from suggestion where id = @target_suggestion::uuid returning id, caster, target_suggestion, regarding, time;

-- name: DeleteVoteByCasterAndSuggestion :one
delete from vote where caster = $1 and target_suggestion = $2;
