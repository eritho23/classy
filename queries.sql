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

-- name: GetGroupByPersonUid :one
select
  grp.uid as group_uid
from
  grp
  inner join person on person.grp = grp.uid
where
  person.uid = $1
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

-- name: UpdatePersonPasswordHashAndDeleteSessionsByPersonUid :exec
with
  updated as (
    update person p
    set
      password_hash = $1,
      password_last_changed = $2
    where
      p.uid = $3
    returning
      p.uid as uid
  )
delete from session s using updated u
where
  s.person = u.uid;

-- name: GetPersonByUid :one
select
  uid,
  username,
  grp
from
  person
where
  uid = $1
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
      regarding = person.uid
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

-- name: GetSuggestionByUidInGroupRegarding :one
select
  suggestion.uid,
  suggestion.suggester,
  suggestion.regarding,
  suggestion.suggestion,
  suggestion.motivation
from
  suggestion
  inner join person as suggester on suggester.uid = suggestion.suggester
  inner join person as regarding_person on regarding_person.uid = suggestion.regarding
where
  suggestion.uid = @suggestion_uid
  and suggestion.regarding = @regarding_uid
  and suggester.grp = @group_uid
  and regarding_person.grp = @group_uid
limit
  1;

-- name: UpdateSuggestion :exec
update suggestion
set
  suggestion = $1,
  motivation = $2
where
  uid = $3;

-- name: GetSuggestionsByRegardingUserInGroup :many
select
  suggestion.uid,
  suggester,
  suggestion.regarding,
  suggestion,
  motivation,
  person.username as suggester_username,
  (
    select
      count(*)
    from
      vote
    where
      target_suggestion = suggestion.uid
  ) as number_of_votes,
  requester_vote.uid as requester_vote_uid,
  requester_vote.target_suggestion as requester_vote_target_suggestion
from
  suggestion
  inner join person on person.uid = suggestion.suggester
  inner join person regarding_person on regarding_person.uid = suggestion.regarding
  left join vote as requester_vote on requester_vote.target_suggestion = suggestion.uid
  and requester_vote.caster = @caster
where
  suggestion.regarding = @regarding_uid
  and person.grp = @group_uid
  and regarding_person.grp = @group_uid
order by
  number_of_votes desc;

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

-- name: GetVoteByUidInGroupRegardingSuggestion :one
select
  vote.uid,
  vote.caster,
  vote.target_suggestion,
  vote.regarding,
  vote.time
from
  vote
  inner join suggestion on suggestion.uid = vote.target_suggestion
  inner join person as suggester on suggester.uid = suggestion.suggester
  inner join person as regarding_person on regarding_person.uid = suggestion.regarding
where
  vote.uid = @vote_uid
  and vote.target_suggestion = @suggestion_uid
  and vote.regarding = @regarding_uid
  and suggestion.regarding = @regarding_uid
  and suggester.grp = @group_uid
  and regarding_person.grp = @group_uid
limit
  1;

-- name: DeleteVoteByUid :exec
delete from vote
where
  uid = $1;

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
  person.username,
  person.password_last_changed
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

-- name: GetGroupAndPersonPartOfGroupByGroupUid :one
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

-- name: ExistsSuggestionOnTargetByUserById :one
select
  exists (
    select
      1
    from
      suggestion
      inner join person on suggestion.suggester = person.uid
      inner join grp on grp.uid = person.grp
    where
      suggester = @suggester_uid
      and regarding = @regarding_uid
      and grp.uid = @group_uid
  ) as suggestion_exists;

-- name: SimpleCompleteChallengeByUid :exec
update challenges
set
  completed_by = @person_uid
where
  uid = @challenge_uid;

-- name: SimpleUncompleteChallengeByUid :exec
update challenges
set
  completed_by = null
where
  uid = @challenge_uid;

-- name: GetAllChallenges :many
select
  uid,
  description,
  assigned_number,
  points,
  extra_points_available,
  extra_points_received,
  completed_by,
  batch_number
from
  challenges
order by
  batch_number asc,
  assigned_number asc;

-- name: GetTotalPoints :one
select
  coalesce(sum(points + extra_points_received), 0)::bigint as sum
from
  challenges
where
  completed_by is not null;

-- name: CompleteChallengeWithExtraPoints :exec
update challenges
set
  completed_by = @person_uid,
  extra_points_received = @extra_points
where
  uid = @challenge_uid;

-- name: GetAllChallengesWithCompleter :many
select
  c.uid as challenge_uid,
  c.description,
  c.assigned_number,
  c.points,
  c.extra_points_available,
  c.extra_points_received,
  c.completed_by,
  c.batch_number,
  p.username as completer_username
from
  challenges c
  left join person p on c.completed_by = p.uid
order by
  c.batch_number asc,
  c.assigned_number asc;
