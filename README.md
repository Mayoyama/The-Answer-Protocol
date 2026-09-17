*This project has been created as part of the 42 curriculum by speterse.*

## Description

TAP (The Answer Protocol) is a shared-world, TCP-based text adventure: a Go server hosts a persistent world (rooms, items, NPCs) that multiple clients connect to over a line-based text protocol (RFC 42TAP), with real-time chat, movement, grouping, and item interaction. This repo currently contains the **server** and a **CLI client**.

## Instructions

See **Building and Running** below.

## Architecture

- **Dispatcher-based design.** Two dispatch layers: `handleLogin` handles the pre-authentication state (CONNECT/QUIT only, with its own token-bucket spam guard), then once a player has an identity, `commandDispatch` takes over as the central command router for all in-game commands (LOOK, MOVE, CHAT, GROUP, TAKE, DROP, TALK, ATTACK, QUEST, ACCEPT, STATUS, INVENTORY, QUESTS, WHO, QUIT). `GROUP` and `CHAT` each have their own sub-dispatcher (`groupFuncDispatcher`, `chatDispatcher`) for their subcommands/scopes.
- **Concurrency model.** One goroutine per accepted TCP connection (`handleTCPConn`), reading line-by-line with a 5-minute idle read deadline. Shared state is protected with fine-grained mutexes rather than a single global lock: each `Player`, `Zone`, and `Group` has its own mutex, plus package-level mutexes for the top-level registries (`onlinePlayers`, `zones`, `parties`, `items`, `npcs`, the softban/rate-limit maps). Broadcasts (chat, zone enter/leave, group events) iterate the relevant map while holding its mutex and write directly to each player's `net.Conn`.
- **Rate limiting / abuse handling** (`security.go`): a per-player token-bucket throttle (refill 0.75/sec, capacity 8) gates command frequency; falling below 1 token starts a 5-minute per-player timeout (`ERR 750`, escalating warnings) that, after 10 repeated violations, becomes a hard IP-level ban (`ERR 760`, 20–30 min) enforced at accept-time before the connection is even handed to `handleTCPConn`. IP bans are keyed on the bare host (port stripped via `net.SplitHostPort`), not `RemoteAddr().String()`, so a client can't dodge a ban by reconnecting on a new ephemeral port. This satisfies RFC §9.4's "chat message frequency" resource-limit recommendation.

## CLI Client

**Command interface choice**

The subject (V.3) allows two approaches: passing user input straight through as RFC protocol syntax, or building a translation layer that maps friendlier commands to protocol packets.

We went with the first option — the CLI sends what the user types directly to the server as-is (e.g. typing `MOVE north` sends `MOVE north` verbatim). The only exceptions are the initial `CONNECT <username>` prompt on startup and `QUIT` on disconnect/Ctrl+C/Ctrl+D, which the client sends automatically. No parsing, aliasing, or friendlier syntax is implemented on top of raw protocol commands.

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

**QUEST command — reward shape deviation**

The subject PDF's example QUEST response encodes `reward` as a bare string:

```
S: OK {"quest_id": "fetch_herbs", "description": "Bring me 3 healing herbs", "reward": "gold_coin", "status": "available"}
```

Our server returns `reward` as a nested object instead of a string, since a quest's reward can be a key item, gold, or both — a single string can't hold that combination without the client having to parse it back apart. `reward` is `null` when a quest has no reward (e.g. a step in a chain with no direct payout).

```go
type QuestResponse struct {
    QuestID     string  `json:"quest_id"`
    Description string  `json:"description"`
    Reward      *Reward `json:"reward"`
    Status      string  `json:"status"`
}

type Reward struct {
    KeyItem string `json:"key_item,omitempty"`
    Money   int    `json:"gold,omitempty"`
}
```

Example from our server:

```
C: QUEST george
S: OK {"quest_id": "quest.bread_for_kassandra", "description": "George seems to need help with his daily chores.", "reward": {"key_item": "rusty_dagger", "gold": 50}, "status": "available"}
```

**Custom command — `ACCEPT`**

RFC 42TAP §6.1.2 defines `QUEST` and `QUESTS` as the only mandated quest commands, and explicitly leaves acceptance mechanics — and any further commands ("e.g. `COMPLETE_QUEST`, `ABANDON_QUEST`, or similar") — to the implementer.

We split quest acceptance out into its own command, `ACCEPT`, rather than folding it into `QUEST`. This keeps `QUEST` a pure, side-effect-free query that matches the RFC's example exactly (`status: "available"`, no player state changed), while `ACCEPT` is the one command that actually commits the player to a quest — adding it to their tracked quest list with `status: "active"`. Both commands resolve the same NPC-eligibility check (`NPC.getNPCQuest`): the first quest that NPC offers which the player doesn't already have, and whose prerequisite (if any) is already completed.

```
C: QUEST george
S: OK {"quest_id": "quest.bread_for_kassandra", "description": "George seems to need help with his daily chores.", "reward": {"key_item": "rusty_dagger", "gold": 50}, "status": "available"}

C: ACCEPT george
S: OK {"quest_id": "quest.bread_for_kassandra", "description": "George seems to need help with his daily chores.", "reward": {"key_item": "rusty_dagger", "gold": 50}, "status": "active"}
```

**Open deviation to resolve:** RFC §4.2 states command names are case-insensitive. The current dispatch (`commandDispatch`, `handleLogin`, `groupFuncDispatcher`, `chatDispatcher`) switches on the command string as received, with no case-folding — so e.g. `move north` would currently fail where `MOVE north` succeeds. Either fold the command name to uppercase before dispatch, or explicitly document that only uppercase is accepted as a deviation.

Dynamic item system (unique instances, no duplication on TAKE, made available again on DROP, lookup by ID or display name via `strings.EqualFold`) is implemented in `items.go`.

## Combat System — TODO (not yet implemented)

`Player` has `MaxHP`/`CurrHP`/`Status` fields and `STATUS` reports them, but `ATTACK` is an unimplemented stub (`commandDispatch.go`, `case "ATTACK": //function here`). NPCs carry a `Stats map[string]int` (currently just `hp`) and an `Attackable` bool from world data, but nothing reads them yet. Still to design and document here per the project's required "Design Choice": damage formula, turn order/initiative, counter-attacks, respawn-with-reduced-HP behavior, and any extra commands (DEFEND, FLEE, etc.).

## Quest System

Quest data model (`quests.go`): a `Quest` (`Name`, `QuestID`, `Description`, `Type`, `Steps []QuestAction`, `Reward *Reward`, `Requires`) has a `QuestType` (`Delivery`, `Fetch`, `Battle`), and each of its steps implements a `QuestAction` interface (`ActionType() string`) as one of `TalkToAction`, `EnterAreaAction`, or `BattleAction`. When `world.yaml` is parsed, the concrete step type is picked based on which of `talk_to`/`enter_area`/`battle` is present on that step (`QuestStep.UnmarshalYAML`, `parser.go`). A `Reward` is optional and can carry a key item, gold, or both.

Quests are defined under `world.yaml`'s top-level `quests:` section and attached to their quest-giver NPC by ID in that NPC's own `quests:` list — there's no separate global quest registry; a player reaches a quest only through the NPC offering it.

**Quest chains.** A quest can optionally name a prerequisite via `requires: <quest_key>` (a bare quest key, `""` if none). `ParseYmlData` validates this at load time by walking each quest's `requires` chain: an unknown prerequisite key is rejected, and so is a chain that loops back on itself, at any length — not just a quest requiring itself directly.

**Per-player progress.** Each `Player` has a `Quests map[string]*PlayerQuest` (`quests.go`), tracking `Quest`, `StepIndex`, and `Status` (`Active`/`Completed`) per accepted quest. `NPC.getNPCQuest` picks the first quest that NPC offers which the player doesn't already have, and whose prerequisite (if any) is already `Completed`.

**Commands.** `QUEST` (`checkNPCQuest`) reports an NPC's next eligible quest for the calling player without changing any state. `ACCEPT` (`acceptQuest` — a custom addition, see Protocol Implementation) commits the player to that quest, adding it to `player.Quests` as `Active`. `QUESTS` (`printPlayerQuests`) lists everything the calling player has accepted, showing `progress` (`stepIndex/totalSteps`) only while a quest is still `Active`.

**Still to build:** advancing `StepIndex` when a quest step is actually completed (hooking into `TALK`/`MOVE`/a future battle handler), and granting the reward on completion — no `Player.Gold`-equivalent field exists yet, and key-item granting is deferred.

Example (`world.yaml`):

```yaml
quests:
  bread_for_kassandra:
    name: "A loaf for the needy"
    description: "George seems to need help with his daily chores."
    type: delivery
    requires: ""
    steps:
      - talk_to: "george"
        dialogue: "Hey there! Boy am I glad you're here. Be a champ and take this 'keyitem.Loaf' to Kassandra. Thanks!"
      - talk_to: "kassandra"
        dialogue: "What's this, a loaf of bread? I'm not hungry, but thanks. Take this, I found it while out walking this morning and have no use for it."
    reward:
      key_item: "rusty_dagger"
      gold: 50
```

## World Design

`world.yaml` currently defines 2 rooms (`taverne` ⇄ `town_square`), 4 items (all obtainable), 3 NPCs — `george` (`quest giver`, offering the one quest below), and `bertha`/`kassandra` (both `general`) — and 1 quest (`bread_for_kassandra`, a delivery quest routed through `george` and `kassandra`). No `enemy`-role NPC exists yet. This is below the project's required world size (≥8 interconnected rooms forming at least one loop plus a branch, ≥3 distinct NPC roles, ≥4 distinct items with ≥2 obtainable, ≥2 quests) and needs expanding before submission — the PDF's own example world is explicitly "intentionally tiny" for the same reason.

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
- AI usage: Claude was used during development to run `golangci-lint` against the server (the local environment can't install it directly), diagnose and fix a bug where softban/IP-ban keys included the ephemeral source port (so a banned client could bypass the ban by reconnecting), help design the quest-system's data model and response shapes before implementation, help split quest acceptance out of `QUEST` into the separate `ACCEPT` command once it became clear the two were conflating a report and a state change, identify and fix a cross-zone mutex-ordering deadlock risk in disconnect cleanup and a redundant-recursion bug in the (not-yet-wired-in) map cycle-detection helper, and draft/restructure this README from the existing source and protocol spec.
