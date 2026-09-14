## Protocol Implementation

**WHO command — deviation from RFC 42TAP**

RFC 42TAP section 5.2.2 specifies `WHO` as:

```
Response: OK players=<count>
```

However, the subject PDF's own "Example interactions" section (NPC interaction and inventory, p.12) shows a different response format:

```
C: WHO
S: OK { "room": ["alice", "bob"], "server": 5 }
```

These two source documents disagree. Our server implements the JSON form shown in the subject PDF's example, since it's the canonical example given for the project and it also gives the GUI client a ready source for the room/server player counters required in V.4.

Example from our server:

```
C: WHO
S: OK {"room":["shane"],"server":1}
```

## CLI Client

**Command interface choice**

The subject (V.3) allows two approaches: passing user input straight through as RFC protocol syntax, or building a translation layer that maps friendlier commands to protocol packets.

We went with the first option — the CLI sends what the user types directly to the server as-is (e.g. typing `MOVE north` sends `MOVE north` verbatim). The only exceptions are the initial `CONNECT <username>` prompt on startup and `QUIT` on disconnect/Ctrl+C/Ctrl+D, which the client sends automatically. No parsing, aliasing, or friendlier syntax is implemented on top of raw protocol commands.



*This project has been created as part of the 42 curriculum by speterse.*

> Draft notes, not a final submission. Sections marked **TODO** need information only you have (build tooling, team split, testing steps) or cover work that isn't implemented yet. Written from the server source as of this session — re-generate/update once the CLI, GUI, and world data exist.

## Description

TAP (The Answer Protocol) is a shared-world, TCP-based text adventure: a Go server hosts a persistent world (rooms, items, NPCs) that multiple clients connect to over a line-based text protocol (RFC 42TAP), with real-time chat, movement, grouping, and item interaction. This repo currently contains the **server** only.

## Instructions

See **Building and Running** below.

## Architecture

- **Dispatcher-based design.** Two dispatch layers: `handleLogin` handles the pre-authentication state (CONNECT/QUIT only, with its own token-bucket spam guard), then once a player has an identity, `commandDispatch` takes over as the central command router for all in-game commands (LOOK, MOVE, CHAT, GROUP, TAKE, DROP, TALK, STATUS, INVENTORY, WHO, QUIT). `GROUP` and `CHAT` each have their own sub-dispatcher (`groupFuncDispatcher`, `chatDispatcher`) for their subcommands/scopes.
- **Concurrency model.** One goroutine per accepted TCP connection (`handleTCPConn`), reading line-by-line with a 5-minute idle read deadline. Shared state is protected with fine-grained mutexes rather than a single global lock: each `Player`, `Zone`, and `Group` has its own mutex, plus package-level mutexes for the top-level registries (`onlinePlayers`, `zones`, `parties`, `items`, `npcs`, the softban/rate-limit maps). Broadcasts (chat, zone enter/leave, group events) iterate the relevant map while holding its mutex and write directly to each player's `net.Conn`.
- **Rate limiting / abuse handling** (`security.go`): a per-player token-bucket throttle (refill 0.75/sec, capacity 8) gates command frequency; falling below 1 token starts a 5-minute per-player timeout (`ERR 750`, escalating warnings) that, after 10 repeated violations, becomes a hard IP-level ban (`ERR 760`, 20–30 min) enforced at accept-time before the connection is even handed to `handleTCPConn`. IP bans are keyed on the bare host (port stripped via `net.SplitHostPort`), not `RemoteAddr().String()`, so a client can't dodge a ban by reconnecting on a new ephemeral port. This satisfies RFC §9.4's "chat message frequency" resource-limit recommendation.

## Protocol Implementation

Error codes `201`, `301`, `401`, `402`, the three `404`s (`ITEM_NOT_FOUND`, `ITEM_NOT_IN_INVENTORY`, `NPC_NOT_FOUND`), `405`, `406`, `900`, and `901` are exactly as defined in RFC 42TAP §7.2 and are not altered.

Custom additions beyond the RFC's error table, all under codes the RFC leaves unassigned:

- `202 NAME_TOO_LONG` / `202 NAME_TOO_SHORT` — same code, two conditions (name outside 2–10 runes); RFC only defines `201` for the username-collision case, not length.
- `204 INVALID_CHAR_IN_NAME` — non-printable characters rejected in usernames (RFC §9.2 says servers SHOULD reject/safely handle control characters).
- `403 NOT_GROUP_LEADER` — GROUP INVITE/JOIN restricted to the group leader; not covered by the RFC's group error set (`401`/`402` only).
- `404 PLAYER_NOT_FOUND` — reuses the RFC's `404` category for a case it doesn't enumerate (inviting a nonexistent player to a group).
- `666 INVALID_COMMAND`, `670 INVALID_ARGS`/`670 MISSING_ARGS` (shared code, two conditions) — malformed-command handling required by RFC §9.3, code values are ours.
- `750 EXCESSIVE_INPUT_DETECTED` / `760 SOFTBANNED_FROM_SERVER` — the rate-limiting/softban system described above.
- `825 INTERNAL_ERROR`, `878 YAML_ERROR`, `880 JSON_ERROR`, `905 ALREADY_CONNECTED`, `911 INBOUND_CONNECTION_FAILURE` — internal/operational errors not addressed by the protocol.

**Open deviation to resolve:** RFC §4.2 states command names are case-insensitive. The current dispatch (`commandDispatch`, `handleLogin`, `groupFuncDispatcher`, `chatDispatcher`) switches on the command string as received, with no case-folding — so e.g. `move north` would currently fail where `MOVE north` succeeds. Either fold the command name to uppercase before dispatch, or explicitly document that only uppercase is accepted as a deviation.

Dynamic item system (unique instances, no duplication on TAKE, made available again on DROP, lookup by ID or display name via `strings.EqualFold`) is implemented in `items.go`.

## Combat System — TODO (not yet implemented)

`Player` has `MaxHP`/`CurrHP`/`Status` fields and `STATUS` reports them, but `ATTACK` and `QUEST` are unimplemented stubs (`commandDispatch.go`, `case "ATTACK": //function here`). NPCs carry a `Stats map[string]int` (currently just `hp`) and an `Attackable` bool from world data, but nothing reads them yet. Still to design and document here per the project's required "Design Choice": damage formula, turn order/initiative, counter-attacks, respawn-with-reduced-HP behavior, and any extra commands (DEFEND, FLEE, etc.).

## Quest System — TODO (not yet implemented)

`Quest` is an empty struct (`quests.go`) and `NPC.Quests` is never populated; `QUEST`/`QUESTS` commands are unimplemented stubs. Still to design: quest progression/state tracking, completion validation, and rewards, to document here once built.

## World Design

`world.yaml` currently defines 2 rooms (`taverne` ⇄ `town_square`), 4 items (all obtainable), and 3 NPCs, all role `general` except `kassandra` (`quest giver`) — no `enemy`-role NPC yet, and no quests defined on any NPC. The project requires at least 8 interconnected rooms forming one or more loops (plus an optional branch), 3 distinct NPC roles, 4 distinct items (≥2 obtainable), and 2 simple quests — the current world file is below that bar and needs expanding before submission (the PDF's own example world is explicitly "intentionally tiny" for the same reason).

## Server Logging

Structured JSON logging via `log/slog` (`slog.NewJSONHandler`, written to stderr), with `INFO`/`WARN`/`ERROR` levels. Every command handler logs its outcome with player name, command, and relevant args/state; connects, disconnects (graceful, idle-timeout, and error paths), and abuse events (spam timeouts, softbans, connection-flood detection) are all logged with the remote address. `PLAYER_CLEANUP_COMPLETE` and per-zone leave events are logged on disconnect/cleanup. JSON output plus slog's built-in timestamps satisfy the "structured, timestamped, parseable" logging requirement.

## Group Contributions — TODO

*(fill in per member: server / CLI / GUI / world design responsibilities)*

## Building and Running — TODO

`go.mod` targets Go 1.24, single dependency `gopkg.in/yaml.v3`. Document your actual build tool (Makefile targets used so far include at least `run-client`) here: install/dependency step, `run-server`, `run-client`, `run-client-gui`, `lint`, `clean`.

## Testing — TODO

*(describe how to exercise multiplayer behavior — e.g. multiple concurrent CLI connections in separate terminals — plus combat and quest mechanics once those exist)*

## Resources

- RFC 42TAP (attached protocol spec) — primary reference for commands, events, and error codes.
- AI usage: Claude was used during development to run `golangci-lint` against the server (the local environment can't install it directly), diagnose and fix a bug where softban/IP-ban keys included the ephemeral source port (so a banned client could bypass the ban by reconnecting), and draft this README's technical sections from the existing source and protocol spec.
