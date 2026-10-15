*This project has been created as part of the 42 curriculum by speterse.*

<a id="top"></a>

# TAP — The Answer Protocol

## Table of Contents

- [Description](#description)
- [Instructions](#instructions)
- [Architecture](#architecture)
- [CLI Client](#cli-client)
- [GUI Client](#gui-client)
- [Protocol Implementation](#protocol-implementation)
- [Combat System](#combat-system)
- [Quest System](#quest-system)
- [World Design](#world-design)
- [Server Logging](#server-logging)
- [Group Contributions](#group-contributions)
- [Building and Running](#building-and-running)
- [Testing](#testing)
- [Resources](#resources)

## Description

TAP (The Answer Protocol) is a shared-world, TCP-based text adventure. A Go server hosts a persistent world (rooms, items, NPCs) that multiple clients connect to over a line-based text protocol (RFC 42TAP), with real-time chat, movement, groups and item interaction, plus turn-based combat, chained quests with key items, and a healer NPC.

The project follows a 42 school assignment (referred to below as "the subject"). The subject comes with a protocol specification, **RFC 42TAP**, which defines the commands, replies, events and error codes. It requires implementations to follow the RFC strictly, to document any deviation, and to keep clients interchangeable between different groups' servers. Where this README mentions a subject section (e.g. V.4), it refers to the assignment's own chapters.

The repository contains the **server**, a **CLI client** (Go and Rust versions) and a **GUI client** (Go with Fyne, in progress; design in [`TAP_GUI_HUD_FSD.md`](TAP_GUI_HUD_FSD.md)). An example session is shown under [CLI Client](#cli-client).

[↑ Back to top](#top)

## Instructions

Requirements: Go, Rust/Cargo for the Rust CLI, and a C compiler for the GUI client (Fyne uses cgo); details under [Building and Running](#building-and-running). From the repository root:

1. `make install` downloads the dependencies and builds the server, both CLI clients and the GUI client.
2. `make run-server` starts the server on TCP port 4242.
3. In another terminal, start a client: `make run-client` (Rust CLI), `make run-go-client` (Go CLI) or `make run-client-gui` (GUI).

`make lint` and `make clean` cover all components.

[↑ Back to top](#top)

## Architecture

### Dispatcher-based design

- `handleLogin` handles the pre-authentication state (`CONNECT`/`QUIT` only, with its own token-bucket spam guard).
- After login, `commandDispatch` routes every in-game command. `CHAT` and `GROUP` have their own sub-dispatchers (`chatDispatcher`, `groupFuncDispatcher`) for scopes and subcommands.

### Concurrency model

- One goroutine per TCP connection (`handleTCPConn`), reading line by line with a 15-minute idle deadline.
- Fine-grained mutexes instead of one global lock: each `Player`, `Zone`, `Group` and `NPC` has its own, plus one per top-level registry (`onlinePlayers`, `zones`, `parties`, `items`, `npcs`, the ban/rate-limit maps). `keyItems` is read-only after loading, so it needs none.
- Broadcasts iterate the relevant map under its mutex and write directly to each player's `net.Conn`.
- Each fight runs in its own goroutine (`processBattle`, `battle.go`). The player's connection goroutine sends one signal per `ATTACK` and waits for that round's result, so rounds are strictly sequential and fight state is only touched by the fight goroutine.

### Rate limiting / abuse handling

Implemented in `security.go`, covering the resource limits recommended in RFC §9.4 (message length, connection count, message frequency):

- **Command throttle:** a per-player token bucket (refill 1.1/s, capacity 12). An empty bucket starts a 1-minute timeout (`ERR 750`). Each input during the timeout is a violation; the 10th becomes a 20-minute IP ban (`ERR 760`).
- **Query bucket:** `LOOK`, `WHO`, `STATUS` and `GOLD` use a separate bucket (refill 0.9/s, capacity 10), so polling doesn't drain the main one. An empty bucket answers `ERR 747` with no penalty; about 20 rejections within 10 seconds start the 1-minute timeout. During a timeout, a bare `QUIT` and queries with tokens left are still answered.
- **Connection flooding:** 5 connections from one host within 5 seconds give a 20–30 minute ban (`trackConnCount`). Bans are keyed on the bare host (port stripped), so reconnecting on a new port doesn't dodge them.
- **Line length:** lines over 1024 bytes get `ERR 675 MESSAGE_TOO_LONG` without using tokens; the 5th within 5 minutes closes the connection. A line over 64 KB closes it at once.

[↑ Back to top](#top)

## CLI Client

Two versions in `CLI/` behave the same way: Rust (`src/main.rs`, `make run-client`) and Go (`cli.go`, `make run-go-client`). Both connect to `localhost:4242`, ask for a username and send `CONNECT <username>`, and keep printing server events while waiting for input.

**Command interface choice (subject V.3):** the subject allows either raw protocol commands or a friendlier translation layer. These clients pass input straight through as RFC protocol syntax (`MOVE north` is sent as `MOVE north`); commands are case-insensitive. The client only sends `CONNECT <username>` at startup and `QUIT` on Ctrl+C or Ctrl+D by itself.

### Example session

Typed lines are lowercase; everything else is printed by the server (Go CLI):

```
OK hello proto=1

Enter a new username between 2 and 10 characters:
speterse
OK connected

EVT STATS players=1
look
OK {"room":{"id":"zone.taverne","name":"Taverne","description":"A dim, smoky room with a long wooden bar and a crackling hearth.","exits":{"east":"zone.fountain_square"}},"players":["speterse"],"items":["item.dented_tankard","item.old_lute"],"npcs":["npc.george"]}
examine dented tankard
ERR 670 INVALID_ARGS
examine item dented tankard
OK {"name":"Dented Tankard","description":"A well-used tankard. Smells of dark ale.","obtainable":true}
take dented tankard
OK taken=item.dented_tankard
talk george
OK {"npc":"npc.george","dialogue":"Welcome, traveler. Mind the floorboards."}
quest george
OK {"quest_id":"quest.a_loaf_for_the_needy","description":"George seems to need help with his daily chores.","reward":{"key_item":"key_item.rusty_dagger"},"status":"available"}
quests
OK [{"quest_id":"quest.a_loaf_for_the_needy","status":"active","progress":"0/2"}]
talk george
OK {"npc":"npc.george","dialogue":"Hey there! Be a champ and take this [keyitem.Loaf] to Kassandra for me, would you? Thanks! (You receive Loaf of Bread)"}
move east
OK room=zone.fountain_square
talk kassandra
OK {"npc":"npc.kassandra","dialogue":"What's this, a loaf of bread? So George thinks to woo me with crumbs... (looks into your eyes...) Ah, so you're the one... This found its way to me this morning but now it has found its true owner."}
speterse receives Rusty Dagger.
who
OK {"room":["speterse"],"server":1}
status
OK {"hp":100,"max_hp":100,"status":"healthy"}
```

[↑ Back to top](#top)

## GUI Client

The GUI client (`GUI/`, Go with Fyne v2.8.1) is in progress. Its design (layout, elements, which subject requirement each one covers, refresh plan) is in [`TAP_GUI_HUD_FSD.md`](TAP_GUI_HUD_FSD.md).

- **Design:** one router goroutine owns the connection and all connection state; widget callbacks only send requests on channels, and widgets are updated through `fyne.Do`. `TAPResponsesParser.go` turns each server line into a typed reply.
- **Login:** the name is trimmed and checked (2–10 characters) before anything is sent; server errors (e.g. `ERR 201 NAME_IN_USE`) are shown under the Connect button. A lost connection returns to the login screen.
- **Built so far:**
  - a pixel-art background image for each room
  - a top row with an HP bar, the room name, and counters for players online and players in the room
  - the player's status and gold
  - a LOOK button with a room details panel (description, players, NPCs, exits, items); LOOK is also polled every 3.5 seconds, so the panel stays current
  - items in the room are taken by clicking them; the inventory list drops an item when clicked and updates right after each take or drop
  - a battle log that opens during a fight (one entry per round, a divider per fight) and hides 5 seconds after it ends; a Battle Log button shows it again
  - direction buttons for moving, disabled during a rate-limit timeout (`ERR 750`)
  - INVENTORY, KEYITEMS, WHO and QUESTS buttons, with results in the log
  - separate chat and log windows; the chat window also shows NPC dialogue (in either TALK reply format) and fight announcements, and each line starts with an emoji showing its kind
  - a text field for typing raw protocol commands

  Not built yet: the RPG menu (Talk, Attack, Quest, Group, Chat), the STATUS and Quit buttons, the Flee entry and Show/Hide UI.
- **Known limitation:** replies don't name the command they answer, and an empty list (`OK []`) looks the same for INVENTORY, KEYITEMS and QUESTS. The GUI matches it to the last of these three that was sent, so if two different ones are sent before the first reply arrives, the "None" line can name the wrong command.

[↑ Back to top](#top)

## Protocol Implementation

All RFC 42TAP commands and events are implemented. Because the subject requires strict RFC compliance with every deviation documented, this section lists where the server deviates from or adds to the RFC.

### Error codes

All errors are sent as `ERR <code> <MESSAGE>`. **RFC** codes are used exactly as the RFC defines them; **Server** codes are additions of this server.

| Code | Message | Source | Meaning |
|---|---|---|---|
| 201 | `NAME_IN_USE` | RFC | Username already taken |
| 202 | `NAME_TOO_LONG` / `NAME_TOO_SHORT` | Server | Username outside 2–10 characters |
| 204 | `INVALID_CHAR_IN_NAME` | Server | Username with a non-printable character or a space, a reserved name (`OK`, `ERR`, `EVT`, `ADMIN`, `SERVER`, `SYSTEM`, `STAFF`) or a prefix that imitates a server line or ID (`<`, `{`, `[`, `(`, `npc.`, `item.`, `zone.`, `key_item.`) |
| 301 | `NO_EXIT` | RFC | No exit in that direction |
| 401 | `NOT_IN_GROUP` | RFC | Group action without being in a group |
| 401 | `NOT_GROUP_LEADER` | Server | `GROUP INVITE` by a non-leader, or `GROUP JOIN` with no such group |
| 402 | `ALREADY_IN_GROUP` | RFC | Player is already in a group |
| 403 | `ITEM_NOT_OBTAINABLE` | Server | `TAKE` on fixed scenery |
| 404 | `ITEM_NOT_FOUND` | RFC | Item not in the room |
| 404 | `ITEM_NOT_IN_INVENTORY` | RFC | Item not in the player's inventory |
| 404 | `NPC_NOT_FOUND` | RFC | NPC not in the room |
| 404 | `PLAYER_NOT_FOUND` | Server | Invited player doesn't exist |
| 405 | `NPC_NOT_HOSTILE` | RFC | NPC can't be attacked |
| 405 | `NPC_CURRENTLY_OCCUPIED` | Server | NPC is already fighting another player |
| 406 | `NO_QUEST_AVAILABLE` | RFC | NPC has no quest for the player |
| 666 | `INVALID_COMMAND` | Server | Unknown command (RFC §9.3 asks for an error reply to malformed input) |
| 666 | `COMMAND_NOT_AVAILABLE_IN_COMBAT` | Server | `MOVE`, `TAKE`, `DROP` or `QUEST` during a fight |
| 670 | `INVALID_ARGS` / `MISSING_ARGS` | Server | Unexpected or missing arguments |
| 675 | `MESSAGE_TOO_LONG` | Server | Line longer than 1024 bytes |
| 747 | `EXCESSIVE_INPUT_DETECTED: TIMEOUT_WARNING` | Server | Query rejected, query bucket empty; no timeout |
| 750 | `EXCESSIVE_INPUT_DETECTED: TIMEOUT_APPLIED` | Server | 1-minute timeout; replies during it end with `Time remaining: <time>` |
| 760 | `SOFTBANNED_FROM_SERVER` | Server | IP temporarily banned |
| 825 | `INTERNAL_ERROR` | Server | Unexpected server-side failure |
| 880 | `JSON_ERROR` | Server | A JSON reply couldn't be built (logged; the client gets `825`) |
| 900 | `CONNECTION_FAILED` | RFC | Connection failed (sent to players when the server shuts down) |
| 901 | `SEND_FAILED` | RFC | Message transmission failed (a group invite couldn't be delivered to the invited player) |
| 905 | `ALREADY_CONNECTED` | Server | `CONNECT` while already logged in |
| 911 | `INBOUND_CONNECTION_FAILURE` | Server | Accepting a connection failed (log only) |

### WHO and TALK: RFC vs subject PDF

The RFC and the subject PDF's example interactions (p.12) disagree on two replies. In both cases this server follows the PDF:

| Command | RFC 42TAP | Subject PDF | This server |
|---|---|---|---|
| `WHO` | `OK players=<count>` | `OK { "room": ["alice", "bob"], "server": 5 }` | `OK {"room":["alice"],"server":1}` |
| `TALK` | `OK Welcome to my bakery!` | `OK { "npc": "guard", "dialogue": "Stay safe, traveler." }` | `OK {"npc":"npc.george","dialogue":"Welcome, traveler. Mind the floorboards."}` |

- **WHO:** the JSON form gives clients both player counters required by the subject (V.4) in one reply.
- **TALK:** the JSON form makes dialogue recognisable by its own shape and names the speaking NPC (by ID). Everything the NPC says is in `dialogue`, including the healer and quest suffixes below. The GUI client accepts both forms.

### QUEST, QUESTS and GROUP

- **`QUEST <npc>`** offers and accepts in one step (RFC §6.1.2 leaves acceptance to the implementer): the NPC's next eligible quest is added to the player's list and returned with `"status":"available"`; `ERR 406` if there is none. `reward` is an object instead of the PDF's string, because a reward can be a key item, gold or both, and is `null` when there is none:
  ```
  C: QUEST george
  S: OK {"quest_id":"quest.a_loaf_for_the_needy","description":"George seems to need help with his daily chores.","reward":{"key_item":"key_item.rusty_dagger"},"status":"available"}
  ```
- **`QUESTS`** adds an optional `quest_items` field (`"held/needed"`) while the current step needs key items:
  ```
  S: OK [{"quest_id":"quest.a_loaf_for_the_needy","status":"completed"},{"quest_id":"quest.arms_dealer","status":"active","progress":"1/2","quest_items":"1/2"}]
  ```
- **`GROUP`:** the group ID is `PT-<leader>`. `GROUP JOIN <leader>` replies `OK group=PT-<leader>`; `GROUP CREATE` replies `OK group=NEWPT-<leader>`, so a client can tell a created group from a joined one. After `GROUP LEAVE`'s `OK`, the player also gets `EVT GROUP LEAVE <own name>`. A leader leaving disbands the group, and every other member gets `EVT GROUP LEAVE` for the leader and for themselves.

### Custom commands

RFC §6.1.1 and §6.1.2 leave extra combat and quest commands to the implementer:

| Command | Reply |
|---|---|
| `FLEE` | Ends the current fight (`OK battle ended`) with no reward and no respawn; the player keeps their HP. `ERR 666` outside a fight. |
| `GOLD` | `OK <amount> GOLD`. Gold comes from quest rewards. |
| `KEYITEMS` | The held key items as a JSON array, like `INVENTORY` (e.g. `OK ["key_item.rusty_dagger"]`). |
| `EXAMINE <NPC\|ITEM\|KEYITEM> <name>` | `OK {"name":…,"description":…}`. NPCs must be in the room and add `npc_role`; items must be in the room or inventory and add `obtainable`; key items must be held. The scope word is required (`ERR 670` otherwise). |

```
C: EXAMINE ITEM dented tankard
S: OK {"name":"Dented Tankard","description":"A well-used tankard. Smells of dark ale.","obtainable":true}
```

### Extra output and events

- **Fight announcements** reuse `EVT ROOM CHAT`, with the NPC's ID in double angle brackets as the sender: `EVT ROOM CHAT <<npc.cave_bat>> Skree! (alice commenced a new battle.)`. A player name can't start with `<`, so they can't be imitated.
- **`EVT RESPAWN room=<room-id>`** (e.g. `room=zone.chapel`) is sent to a player who loses a fight, after the move.
- **Inside a TALK reply's `dialogue`:** a healer's heal line ends with `(You feel a warm glow)`; a quest step's `missing_items_dialogue` replaces the normal line; received key items are appended as `(You receive Loaf of Bread)`.
- **Quest messages** (an `enter_area` step's message, reward lines) are plain-text lines sent after the command's `OK`.
- **`ATTACK`** returns more fields than the RFC example (see Combat System).
- **`EVT GROUP LEAVE`** is also sent to the player who leaves (see GROUP above).
- **`EVT STATS players=<count>`** is sent to everyone on every login and disconnect; a new player gets it right after `OK connected`.

### Other notes

- **Movement:** exits use six directions (`north`, `east`, `south`, `west`, `up`, `down`). Any other word, or a missing exit, returns `ERR 301 NO_EXIT`. In `LOOK`, exit values are room IDs in the same form as `room.id` and `OK room=` (`"exits": {"north": "zone.chapel"}`).
- **Empty lists** in `LOOK` and `WHO` are sent as `[]`, as in the PDF's examples.
- **Case-insensitive input:** commands (RFC §4.2), `CHAT` scopes, `GROUP` subcommands, directions, and item/NPC names.
- **Items:** unique instances, removed from the room on `TAKE`, back on `DROP`, found by ID or display name. Key items are kept outside this system (see Quest System).

[↑ Back to top](#top)

## Combat System

`ATTACK <npc>` against an `enemy` NPC in the room starts a **one-on-one, instanced fight**. The NPC is marked occupied (others get `ERR 405 NPC_CURRENTLY_OCCUPIED`), the room is told the fight has started, and each further `ATTACK` from that player resolves one round. Other players only see the start and end announcements. `FLEE` is the only additional combat command; there is no `DEFEND`.

### Stats and rounds

- Players start with 100 HP, strength 7, battle skill 7 and dexterity 10. The Ornate Dagger quest reward adds +2 battle skill. NPC stats come from `world.yaml` (`hp`, `str`, `battle_skill`, `dex`).
- Each round, both sides roll a d6 for initiative (a tie goes to the player). Initiative only decides who strikes first; both sides attack, unless the first hit brings the other to 0 HP.
- Damage is `strength + roll(1..battle_skill)` (8–14 for a player).
- An attack is dodged completely if a 1–100 roll is not higher than the defender's dexterity (dex 10 = 10% dodge).

### Ending a fight

- **Win:** the NPC is "defeated", not killed. Its fight HP was a per-fight copy, so it is available again at full HP. The room is told who won, and a quest waiting on that fight advances.
- **Loss:** the player respawns in the chapel with half their max HP (rounded up) and receives `EVT RESPAWN room=zone.chapel`. A loss never fails a quest.
- **`FLEE` or disconnect:** the fight ends early and the room is told.

### Status and healing

- During a fight the status is `engaged`; afterwards it is recalculated from HP: `injured` (≤ 30%), `weakened` (≤ 60%), otherwise `healthy`. If it can't be calculated (max HP of 0, which shouldn't happen), it becomes `unknown` and a warning is logged.
- `TALK` to a `healer` while below max HP restores full HP; the reply doesn't show HP, so players check `STATUS`.

### Design choices

- **Instanced fights** keep rounds sequential and fight state owned by one goroutine, with no interference from other players.
- **Both sides attack every round**, so a lucky initiative roll helps without deciding the fight. Dexterity makes enemies feel different: the Cave Bat is weak but hard to hit.
- **Defeated, not killed, and respawn at half HP:** enemies stay available to everyone, and losing is a setback rather than a failure.
- **Balance** was tuned with a fight simulation: a full-HP player beats the Cave Bat (losing about 24 HP on average), and the Barbarian (90 HP) is the tougher fight.

### `ATTACK` response

The RFC example's `attacker_hp`, `target_hp`, `damage` and `status` are kept, with the round's details added. `status` is the player's own status rather than the RFC example's `"combat"`:

```
C: ATTACK cave bat
S: OK {"attacker_init_roll":4,"target_init_roll":2,"attacker_hp":95,"attacker_max_hp":100,"target_hp":25,"target_max_hp":36,"target_dodged":false,"damage":11,"attacker_dodged":false,"damage_received":5,"status":"engaged"}
```

[↑ Back to top](#top)

## Quest System

Quests are defined in `world.yaml` and offered only through their quest-giver NPC. A quest has a type (delivery, fetch or battle), an ordered list of steps, an optional reward (a key item, gold or both) and an optional prerequisite (`requires`). Prerequisite chains are checked at load time for unknown quests and loops of any length.

### Progression

`QUEST <npc>` accepts the NPC's first quest that the player doesn't have yet and whose prerequisite is completed. Progress is tracked per player and per connection. Each step advances only when it is the quest's current step, inside the handler for its action:

- **`talk_to`** (`TALK`): the step's dialogue replaces the NPC's normal lines. A step can give key items (`grants_key_items`) or require and take them (`receives_key_items`); without the required items the NPC replies with the step's `missing_items_dialogue`.
- **`enter_area`** (`MOVE`): entering the named room sends the step's message. Respawning there doesn't count.
- **`battle`** (`ATTACK`): winning against the named NPC advances the step; a loss leaves it unchanged.

When the last step completes, the quest is marked completed and the reward is granted, each part announced with a plain-text line (e.g. `alice receives 20 gold.`).

### Key items

Key items are per-player quest items (`key_items:` in `world.yaml`, IDs `key_item.<key>`), listed with `KEYITEMS`. They never appear in rooms and can't be taken or dropped, so every player gets their own copy without breaking the item-uniqueness rule. Two rules apply:

- **Step grants are temporary:** a key item given by a step must be taken back by a later step of the same quest (e.g. George's loaf, delivered to Kassandra).
- **Rewards are kept** and can be required by any other quest: *Arms Dealer* takes the Rusty Dagger and Small Gem (rewards of Kassandra's two quests) and gives back the Ornate Dagger. It can be accepted early but not finished until the player holds both items.

### Example

One quest from `server/world.yaml` (dialogue shortened):

```yaml
a_loaf_for_the_needy:
  name: A loaf for the needy
  description: George seems to need help with his daily chores.
  type: delivery
  requires: ""
  steps:
    - talk_to: george
      dialogue: "Hey there! Be a champ and take this [keyitem.Loaf] to Kassandra for me, would you? Thanks!"
      grants_key_items:
        - "key_item.loaf_of_bread"
    - talk_to: kassandra
      dialogue: "What's this, a loaf of bread? ..."
      receives_key_items:
        - "key_item.loaf_of_bread"
  reward:
    key_item: "key_item.rusty_dagger"
```

Quest data is validated at startup (see [World Design → Validation](#validation)).

[↑ Back to top](#top)

## World Design

A small, light-hearted fantasy town, defined in `server/world.yaml`. New players start in the Taverne; defeated players respawn in the Chapel.

### Rooms (10)

| Room | Exits | NPCs |
|---|---|---|
| Taverne (`taverne`) | east → Fountain Square | George |
| Fountain Square (`fountain_square`) | north → Chapel, east → Old Well, south → Market Square, west → Taverne | Kassandra |
| Chapel (`chapel`) | east → Churchyard, south → Fountain Square | Father Tieu |
| Churchyard (`churchyard`) | south → Old Well, west → Chapel | Osric |
| Old Well (`old_well`) | north → Churchyard, south → Mill, west → Fountain Square, down → Cistern | — |
| Cistern (`cistern`) | up → Old Well | Cave Bat |
| Market Square (`market_square`) | north → Fountain Square, east → Mill, west → Smithy | Bertha |
| Smithy (`smithy`) | east → Market Square | Brannoc |
| Mill (`mill`) | north → Old Well, east → Fallow Field, west → Market Square | Hilde |
| Fallow Field (`fallow_field`) | west → Mill | Barbarian |

Two loops share the Old Well (Fountain Square → Chapel → Churchyard → Old Well, and Fountain Square → Market Square → Mill → Old Well), with three dead-end branches (Cistern, Smithy, Fallow Field). Every room has an NPC or a fixed item.

### NPCs, items and quests

- **NPCs (9, 4 roles):** quest givers George, Kassandra, Bertha and Brannoc; healer Father Tieu; general NPCs Osric and Hilde, who cycle through their lines; enemies the Barbarian (90 HP) and the Cave Bat (36 HP, but dex 30).
- **Items (19, 7 obtainable):** Dented Tankard, Empty Coin Purse, Prayer Candle, Wooden Bucket, Metal Shavings, Handful of Wheat Husks and Field Flower can be taken; the rest (e.g. Weathered Fountain, Anvil, Water Wheel) are fixed scenery. Key items: Loaf of Bread, Rusty Dagger, Small Gem, Ornate Dagger.
- **Quests (4):**
  - *A loaf for the needy* (George): deliver bread to Kassandra. Reward: Rusty Dagger.
  - *Something in the Well* (Kassandra, requires the first quest): find what makes noise in the well. Reward: Small Gem and 20 gold.
  - *A Lesson in Manners* (Bertha): defeat the brute who wrecked her stall. Reward: 10 gold.
  - *Arms Dealer* (Brannoc): bring the Rusty Dagger and Small Gem. Reward: Ornate Dagger (+2 battle skill).

### Validation

The server refuses to start, listing every problem, if `world.yaml` breaks any of these rules:

- **Rooms and map:** at least 8 rooms; every room has a name, a description and an exit; exits use the six directions without duplicates and lead to existing rooms; every room is reachable from the start (breadth-first search) and the map has at least one loop (depth-first search); the starting and respawn rooms exist.
- **Items:** each item is placed only once; items and key items need names and descriptions; no two share a display name (case-insensitive).
- **NPCs:** roles match their data (quest givers have quests; enemies are attackable with `str` and `battle_skill` ≥ 1 and `dex` ≤ 90; healers have a heal line and can't be attacked).
- **Quests:** step targets exist; `battle` steps target an enemy; `talk_to` steps don't target a healer; `enter_area` steps have a message; every key item reference exists; step grants are received back within the same quest; received items are granted somewhere.
- **Minimums:** ≥ 4 items, ≥ 2 obtainable, ≥ 3 NPC roles, ≥ 2 quests.

[↑ Back to top](#top)

## Server Logging

### Format and output

Structured JSON logging with `log/slog`, one object per line on stderr, with `INFO`, `WARN` and `ERROR` levels. Every line has a `time` (RFC 3339), `level` and `msg`, plus attributes such as `player`, `command`, `args` and `remote`:

```json
{"time":"2026-10-09T21:14:03+02:00","level":"INFO","msg":"PLAYER_CONNECTED","remote":"127.0.0.1:51234","player":"alice","args":"alice"}
```

### Event types

- **Commands:** each handler logs its outcome with player, command and arguments; failures use the protocol error as `msg` (e.g. `ERR 301 NO_EXIT`).
- **Replies and events sent:** `SYS_MESSAGE` with the `message` and its `player` or `recipient`.
- **Connections:** `PLAYER_CONNECTED`, `PLAYER_QUIT`, `CONNECTION_CLOSED_BY_CLIENT`, `CONNECTION_CLOSED_SAFELY`, `CONNECTION_IDLE_TIMEOUT_ERROR`, `CONNECTION_READ_ERROR`, `SET_READ_DEADLINE_ERROR`, `PLAYER_CLEANUP_COMPLETE`.
- **Startup and shutdown:** `WORLD_DATA_VALIDATED`, `MAP_CONNECTIVITY_VALIDATED`, `MAP_LOOP_VALIDATED`, `SHUTDOWN_SIGNAL_RECEIVED`, `PERFORMING_CLEANUP`, `SERVER_SHUTTING_DOWN`; world validation problems at `ERROR`.
- **Abuse:** timeouts, bans, oversized lines and connection floods at `WARN`.

### Monitoring and detecting abuse

```
make run-server 2> server.log                  # keep the log in a file
tail -f server.log                             # follow it live
grep '"level":"WARN"' server.log               # warnings only
grep -E 'ERR (675|747|750|760)' server.log     # rate-limit, line-length and ban events
```

- **Command flooding:** `ERR 747` and `ERR 750` in the `message` of `WARN` lines, with `player` and `remote`; `time_remaining` shows how long a timeout lasts, and `"reason":"query bucket flooding"` marks a timeout caused by queries. `ERR 747` is logged for the first rejection in a window and then every 5th.
- **Oversized lines:** `ERR 675 MESSAGE_TOO_LONG` for the first one, and an entry ending in `5TH_STRIKE_CLOSING_CONNECTION` when the connection is closed.
- **Bans:** `ERR 760 SOFTBANNED_FROM_SERVER` with the `remote` address. Several `ERR 750` lines for one player followed by `ERR 760` show a client that ignored its timeout.

[↑ Back to top](#top)

## Group Contributions

TAP is designed for 2–3 learners. As no eligible peers were available, I completed the whole project on my own:

- **Server** (Go): protocol, world loading and validation, combat, quests
- **CLI clients**: Go and Rust versions
- **GUI client** (Go, Fyne): in progress
- **World design**: rooms, NPCs, items, dialogue and quests
- **Documentation**: this README and the GUI design record

[↑ Back to top](#top)

## Building and Running

Requirements:

- Go (the server targets Go 1.27; dependency `gopkg.in/yaml.v3`)
- Rust/Cargo for the Rust CLI (dependencies `anyhow`, `ctrlc`)
- for the GUI, Fyne v2.8.1 (downloaded by `go mod`), which needs cgo: a C compiler and the platform's graphics libraries (see Fyne's Getting Started page)

From the repository root:

| Target | What it does |
|---|---|
| `make install` | Downloads Go modules and builds the server, both CLI clients and the GUI client |
| `make run-server` | Starts the server on TCP port 4242 (reads `server/world.yaml`) |
| `make run-client` | Starts the Rust CLI client |
| `make run-go-client` | Starts the Go CLI client |
| `make run-client-gui` | Starts the GUI client |
| `make lint` | `go vet` on the server, Go CLI and GUI; `cargo check` on the Rust CLI |
| `make clean` | `go clean` / `cargo clean` |

`make -C <server|CLI|GUI> format` runs `gofmt` (and `cargo fmt` for the CLI); `make build-GUI` builds only the GUI. All clients connect to `localhost:4242`, so start the server first. In the CLI, type protocol commands directly (e.g. `LOOK`, `MOVE east`, `TALK george`); `QUIT`, Ctrl+C or Ctrl+D disconnects.

[↑ Back to top](#top)

## Testing

NPCs and items can be named by display name (case-insensitive, e.g. `ATTACK cave bat`) or full ID (`npc.cave_bat`).

### Multiplayer

Run several CLI clients with different names. Check `WHO`, `CHAT ROOM`/`GLOBAL`/`GROUP`, room enter/leave events, groups (`GROUP CREATE`/`INVITE`/`JOIN`/`LEAVE`), and item uniqueness: once one player `TAKE`s an item, the other no longer sees it in `LOOK`, and `DROP` brings it back.

### Combat

Go to the Fallow Field (from the Taverne: east, south, east, east) and `ATTACK barbarian` until the fight ends. During the fight, `MOVE` is refused (`666`) and a second client's `ATTACK barbarian` is refused (`405`). Lose a fight to check the respawn in the chapel at half HP, then `TALK father tieu` and check `STATUS`. Try `FLEE` mid-fight.

### Quests

1. Taverne: `QUEST george`, then `TALK george`; the dialogue ends with `(You receive Loaf of Bread)` and `KEYITEMS` lists the loaf.
2. Fountain Square: `TALK kassandra` hands over the loaf and completes the quest; `QUESTS` shows it as completed.
3. `QUEST kassandra` accepts *Something in the Well*; `TALK kassandra`.
4. `MOVE east` into the Old Well shows the eerie-sounds message.
5. `MOVE down` and `ATTACK cave bat` until it is defeated.
6. Back in Fountain Square, `TALK kassandra` completes the quest; `GOLD` shows 20.
7. Smithy: `QUEST brannoc`, `TALK brannoc` (hint), `TALK brannoc` again hands over the dagger and gem and gives the Ornate Dagger with `+2 BattleSkill`.

*Arms Dealer* can also be accepted early: `TALK brannoc` then gives his missing-items line, and `QUESTS` shows `"quest_items":"0/2"` or `"1/2"`. *A Lesson in Manners* works like the Cave Bat quest, with Bertha and the Barbarian.

### EXAMINE

`EXAMINE ITEM dented tankard` (in the Taverne, also after taking it), `EXAMINE NPC george` (shows `npc_role`), `EXAMINE KEYITEM rusty dagger` before and after handing it to Brannoc (`ITEM_NOT_IN_INVENTORY` afterwards), and `EXAMINE FOO x` (`INVALID_ARGS`).

### GUI

`make run-client-gui` with the server running. A name that is empty or one character long is refused without contacting the server; a name in use shows `ERR 201`; stopping the server brings back the login screen. In game, click an item in the room details to take it and in the inventory to drop it; a second client sees the change within 3.5 seconds. `ATTACK cave bat` in the Cistern opens the battle log. Compare the game screen with [`TAP_GUI_HUD_FSD.md`](TAP_GUI_HUD_FSD.md).

### World validation

Break `world.yaml` on purpose (an exit to a missing room, a direction like `northwest`, a quest step targeting a missing NPC, a healer without `heal_dialogue`, a key item named like an item, a step grant that is never received back) and check that the server refuses to start and lists each problem.

[↑ Back to top](#top)

## Resources

- RFC 42TAP (attached protocol spec): commands, events and error codes
- [`TAP_GUI_HUD_FSD.md`](TAP_GUI_HUD_FSD.md): design record for the GUI client
- Go documentation: [`net`](https://pkg.go.dev/net), [`log/slog`](https://pkg.go.dev/log/slog), [`sync`](https://pkg.go.dev/sync), [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3)
- Fyne: [documentation](https://docs.fyne.io/) and [API reference](https://pkg.go.dev/fyne.io/fyne/v2)
- Rust crates: [`anyhow`](https://docs.rs/anyhow), [`ctrlc`](https://docs.rs/ctrlc)

### AI Usage

AI (Claude) was used during development for:

- Running `golangci-lint` against the server, since it couldn't be installed locally
- Brainstorming the quest data model, the key item design and the reply shapes
- Spotting bugs: a softban bypass (ban keys included the client's port), a cross-zone mutex-ordering deadlock risk, a self-deadlock when a player disconnects mid-fight, lock/unlock pairing in the quest-step hooks, and redundant recursion in the map cycle check
- Suggesting world-building ideas (layout, NPCs, items, draft descriptions and dialogue), which were then edited and chosen by hand
- Simulating fights to balance enemy stats
- Reviewing code: the login path, rate limiting, `EXAMINE`, `KEYITEMS`, `quest_items`, and checking `world.yaml` for broken references
- Checking the server's replies and events against the RFC and the subject PDF
- GUI client: explaining Fyne's API and Go concurrency patterns, and reviewing the connection layer, parser, game screen, battle log and item lists
- Drafting and formatting this README and the GUI design record, and checking them against the subject and the code

[↑ Back to top](#top)
