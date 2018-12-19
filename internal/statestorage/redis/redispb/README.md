# Redis State Storage Protobuffer Modules
These are modules used to read ('unmarshal'), write ('marshal'), and monitor Open Match protobuffer formats directly to/from Redis.

## FAQs
1. Why are there separate implementations for the Frontend objects (Players/Groups, participants in matchmaking) and Backend objects (MatchObjects, which hold profiles and match results)?
We'd like to unify these at some point, but to make a more generic version of this library, we'd really like to depend on golang reflection, which is kind of messy right now for protobuffers.  For now, we'll just focus on having separate implementations to carry us until the situation improves.
