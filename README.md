<a id="top"></a>

*This project has been created as part of the 42 curriculum by speterse.*

# TAP — The Answer Protocol

## Table of Contents

- [Description](#description)
- [Instructions](#instructions)
- [Architecture](#architecture)
- [CLI Client](#cli-client)
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

TAP (The Answer Protocol) is a shared-world, TCP-based text adventure: a Go server hosts a persistent world (rooms, items, NPCs) that multiple clients connect to over a line-based text protocol (RFC 42TAP), with real-time chat, movement, grouping, and item interaction. The server also runs a turn-based combat system, a quest system with chained quests and key items, and a healer NPC. This repo currently contains the **server** and a **CLI client** (in both Go and Rust); the **GUI client** (planned in Go with Fyne) is not yet implemented.

[↑ Back to top](#top)

## Instructions

See **Building and Running** below.

[↑ Back to top](#top)

## Architecture

### Dispatcher-based design
Two dispatch layers: `handleLogin` handles the pre-authentication state (CONNECT/QUIT only, with its own token-bucket spam guard), then once a player has an identity, `commandDispatch` takes over as the central command router for all in-game commands (LOOK, MOVE, CHAT, GROUP, TAKE, DROP, TALK, EXAMINE, ATTACK, FLEE, QUEST, ACCEPT, STATUS, INVENTORY, KEYITEMS, QUESTS, GOLD, WHO, QUIT). `GROUP` and `CHAT` each have their own sub-dispatcher (`groupFuncDispatcher`, `chatDispatcher`) for their subcommands/scopes.

### Concurrency model
One goroutine per accepted TCP connection (`handleTCPConn`), reading line-by-line with a 15-minute idle read deadline. Shared state is protected with fine-grained mutexes rather than a single global lock: each `Player`, `Zone`, and `Group` has its own mutex, plus package-level mutexes for the top-level registries (`onlinePlayers`, `zones`, `parties`, `items`, `npcs`, the softban/rate-limit maps). The `keyItems` registry is only written while `world.yaml` is loaded and is read-only afterwards, so it needs no mutex. Broadcasts (chat, zone enter/leave, group events) iterate the relevant map while holding its mutex and write directly to each player's `net.Conn`.

### Combat goroutines
Each fight runs in its own goroutine (`processBattle`, `battle.go`). The player's connection goroutine sends one signal per `ATTACK` on an unbuffered channel and waits for that round's result on a second channel, so rounds are strictly sequential and fight state is only touched by the fight goroutine. Ongoing fights are tracked in `OngoingBattles` (keyed by player name, own mutex).

### Rate limiting / abuse handling
Implemented in `security.go`: a per-player token-bucket throttle (refill 0.75/sec, capacity 8) gates command frequency; falling below 1 token starts a 5-minute per-player timeout (`ERR 750`, escalating warnings) that, after 10 repeated violations, becomes a hard IP-level ban (`ERR 760`, 20–30 min) enforced at accept-time before the connection is even handed to `handleTCPConn`. IP bans are keyed on the bare host (port stripped via `net.SplitHostPort`), not `RemoteAddr().String()`, so a client can't dodge a ban by reconnecting on a new ephemeral port. This satisfies RFC §9.4's "chat message frequency" resource-limit recommendation.

[↑ Back to top](#top)

## CLI Client

### Two implementations
The CLI client exists in two versions, both in `CLI/`:

- **Rust** (`src/main.rs`): run with `make run-client`
- **Go** (`cli.go`): run with `make run-go-client`

Both behave the same way: they connect to the server on `localhost:4242`, ask for a username and send `CONNECT <username>` automatically, pass every typed line to the server unchanged, and send `QUIT` on Ctrl+C or Ctrl+D.

### Command interface choice

The subject (V.3) allows two approaches: passing user input straight through as RFC protocol syntax, or building a translation layer that maps friendlier commands to protocol packets.

The CLI uses the first option: it sends what the user types directly to the server as-is (e.g. typing `MOVE north` sends `MOVE north` verbatim). The only exceptions are the initial `CONNECT <username>` prompt on startup and `QUIT` on disconnect/Ctrl+C/Ctrl+D, which the client sends automatically. No parsing, aliasing, or friendlier syntax is implemented on top of raw protocol commands.

[↑ Back to top](#top)

## Protocol Implementation

All errors are sent as `ERR <code> <MESSAGE>`.

### RFC 42TAP standard error codes
These are used exactly as defined in the RFC's standard error code table and are not altered:

| Code | Message | Meaning |
|---|---|---|
| 201 | `NAME_IN_USE` | Requested username is already taken |
| 301 | `NO_EXIT` | Invalid movement direction |
| 401 | `NOT_IN_GROUP` | Group operation requires group membership |
| 402 | `ALREADY_IN_GROUP` | Player already belongs to a group |
| 404 | `ITEM_NOT_FOUND` | Requested item is not in the room |
| 404 | `ITEM_NOT_IN_INVENTORY` | Requested item is not in the player's inventory |
| 404 | `NPC_NOT_FOUND` | Requested NPC is not in the room |
| 405 | `NPC_NOT_HOSTILE` | NPC can't be attacked (not an enemy) |
| 406 | `NO_QUEST_AVAILABLE` | NPC has no quest available for the player |
| 900 | `CONNECTION_FAILED` | Connection failed |
| 901 | `SEND_FAILED` | Message transmission failed |

### Server-specific error codes
Additions beyond the RFC's table, using codes the RFC leaves unassigned or reusing an RFC category for a case it doesn't list:

| Code | Message | Meaning |
|---|---|---|
| 202 | `NAME_TOO_LONG` / `NAME_TOO_SHORT` | Username outside 2–10 characters (the RFC only defines `201` for name collisions) |
| 204 | `INVALID_CHAR_IN_NAME` | Username contains non-printable characters (RFC §9.2 says servers should reject or safely handle control characters) |
| 401 | `NOT_GROUP_LEADER` | `GROUP INVITE`/`JOIN` restricted to the group leader (same category as the RFC's `NOT_IN_GROUP`) |
| 403 | `ITEM_NOT_OBTAINABLE` | `TAKE` on fixed scenery that can't be picked up |
| 404 | `PLAYER_NOT_FOUND` | Invited player doesn't exist |
| 405 | `NPC_CURRENTLY_OCCUPIED` | NPC is already in a fight with another player |
| 666 | `INVALID_COMMAND` | Unknown command (malformed-command handling, RFC §9.3) |
| 666 | `COMMAND_NOT_AVAILABLE_IN_COMBAT` | `MOVE`, `TAKE`, `DROP`, `QUEST` and `ACCEPT` are refused during a fight |
| 670 | `INVALID_ARGS` / `MISSING_ARGS` | Unexpected or missing arguments |
| 750 | `EXCESSIVE_INPUT_DETECTED: TIMEOUT_APPLIED` | Rate limit exceeded; temporary timeout (see Architecture) |
| 760 | `SOFTBANNED_FROM_SERVER` | IP temporarily banned after repeated violations (see Architecture) |
| 825 | `INTERNAL_ERROR` | Unexpected server-side failure |
| 878 | `YAML_ERROR` | Reserved for world-data parsing errors (not currently sent to clients) |
| 880 | `JSON_ERROR` | Logged when a JSON response can't be built; the client receives `825` instead |
| 905 | `ALREADY_CONNECTED` | `CONNECT` sent while already logged in |
| 911 | `INBOUND_CONNECTION_FAILURE` | Logged when accepting an incoming connection fails (server log only) |

### WHO command

RFC 42TAP section 5.2.2 specifies `WHO` as:

```
Response: OK players=<count>
```

However, the subject PDF's own "Example interactions" section (NPC interaction and inventory, p.12) shows a different response format:

```
C: WHO
S: OK { "room": ["alice", "bob"], "server": 5 }
```

These two source documents disagree. This server implements the JSON form shown in the subject PDF's example, since it's the canonical example given for the project and it also gives the GUI client a ready source for the room/server player counters required in V.4.

Example from this server:

```
C: WHO
S: OK {"room":["shane"],"server":1}
```

### QUEST command

The subject PDF's example QUEST response encodes `reward` as a bare string:

```
S: OK {"quest_id": "fetch_herbs", "description": "Bring me 3 healing herbs", "reward": "gold_coin", "status": "available"}
```

This server returns `reward` as a nested object instead of a string, since a quest's reward can be a key item, gold, or both — a single string can't hold that combination without the client having to parse it back apart. `reward` is `null` when a quest has no reward (e.g. a step in a chain with no direct payout).

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

Example from this server:

```
C: QUEST george
S: OK {"quest_id": "quest.bread_for_kassandra", "description": "George seems to need help with his daily chores.", "reward": {"key_item": "key_item.rusty_dagger"}, "status": "available"}
```

### QUESTS command

The RFC's `QUESTS` example shows `quest_id`, `status` and `progress`. This server keeps those and adds an optional `quest_items` field (`"held/needed"`) while a quest's current step requires key items, so the player can see how many they still need to bring. The field is left out for all other quests.

```
C: QUESTS
S: OK [{"quest_id":"quest.bread_for_kassandra","status":"completed"},{"quest_id":"quest.arms_dealer","status":"active","progress":"1/2","quest_items":"1/2"}]
```

### Custom commands

RFC 42TAP §6.1.1 and §6.1.2 leave additional combat and quest commands to the implementer. This server adds the following commands; all of them use the standard `OK` / `ERR <code> <MESSAGE>` replies.

#### `ACCEPT`

RFC 42TAP §6.1.2 defines `QUEST` and `QUESTS` as the only mandated quest commands, and explicitly leaves acceptance mechanics — and any further commands ("e.g. `COMPLETE_QUEST`, `ABANDON_QUEST`, or similar") — to the implementer.

Quest acceptance is split out into its own command, `ACCEPT`, rather than folded into `QUEST`. This keeps `QUEST` a pure, side-effect-free query that matches the RFC's example exactly (`status: "available"`, no player state changed), while `ACCEPT` is the one command that actually commits the player to a quest — adding it to their tracked quest list with `status: "active"`. Both commands resolve the same NPC-eligibility check (`NPC.getNPCQuest`): the first quest that NPC offers which the player doesn't already have, and whose prerequisite (if any) is already completed.

```
C: QUEST george
S: OK {"quest_id": "quest.bread_for_kassandra", "description": "George seems to need help with his daily chores.", "reward": {"key_item": "key_item.rusty_dagger"}, "status": "available"}

C: ACCEPT george
S: OK {"quest_id": "quest.bread_for_kassandra", "description": "George seems to need help with his daily chores.", "reward": {"key_item": "key_item.rusty_dagger"}, "status": "active"}
```

#### `FLEE`

`FLEE` (no argument; a player can only be in one fight at a time) ends the player's current fight early (`OK battle ended`). The fight is abandoned with no reward and no respawn; the player keeps their current HP. Outside a fight it returns `ERR 666 INVALID_COMMAND`. RFC 42TAP §6.1.1 lists `FLEE` as an example of an implementer-defined combat command.

#### `GOLD`

`GOLD` (no argument) returns the player's gold balance as `OK <amount> GOLD`. Gold is earned from quest rewards.

#### `EXAMINE`

`LOOK` takes no arguments in the RFC, so item and NPC descriptions from `world.yaml` would otherwise never reach the player. `EXAMINE` (`examine.go`) shows them:

```
EXAMINE <NPC|ITEM|KEYITEM> <name>
```

The first word picks what to look up (case-insensitive); the rest is the name, matched by display name, ID or `world.yaml` key, case-insensitively. The scope word is required: `EXAMINE tankard` without it returns `ERR 670 INVALID_ARGS`.

- `NPC` — the NPC must be in the player's room (`ERR 404 NPC_NOT_FOUND` otherwise).
- `ITEM` — the item must be in the player's room or inventory (`ERR 404 ITEM_NOT_FOUND` otherwise).
- `KEYITEM` — the player must hold the key item (`ERR 404 ITEM_NOT_IN_INVENTORY` otherwise).
- Any other scope word returns `ERR 670 INVALID_ARGS`.

```
C: EXAMINE ITEM tankard
S: OK {"name":"Dented Tankard","description":"A well-used tankard, still smells faintly of ale."}
```

#### `KEYITEMS`

`KEYITEMS` (no argument) lists the key items the player holds, in the same JSON-array format as `INVENTORY`. Key items are kept out of `INVENTORY` on purpose, so that command still lists only world items as the RFC describes (see Quest System → Key items).

```
C: KEYITEMS
S: OK ["key_item.rusty_dagger","key_item.small_gem"]
```

### Events
All RFC 42TAP events are implemented and pushed to clients as they happen:

| Event | Sent when |
|---|---|
| `EVT ROOM PRESENCE ENTER <username>` | A player enters the room: by `MOVE`, by logging in (starting room), or by respawning. Sent to the players already there. |
| `EVT ROOM PRESENCE LEAVE <username>` | A player leaves the room: by `MOVE`, by respawning elsewhere, or by disconnecting. Sent to the players still there. |
| `EVT ROOM CHAT <username> <message>` | `CHAT ROOM`, to the other players in the room |
| `EVT GLOBAL CHAT <username> <message>` | `CHAT GLOBAL`, to all other online players |
| `EVT GROUP INVITE <leader>` | A group leader invites the player |
| `EVT GROUP JOIN <username>` / `EVT GROUP LEAVE <username>` | A player joins or leaves the group, to its members |
| `EVT GROUP CHAT <username> <message>` | `CHAT GROUP`, to the other group members |
| `EVT STATS players=<count>` | The online player count changes (login or disconnect), to every online player. A player who logs in receives it right after `OK connected`, so the login reply always comes first. |

### Combat, healing and quest output

- `ATTACK` returns the RFC's `OK <combat-result>` as a JSON object with more fields than the RFC example (see Combat System).
- Fight start/end announcements reuse the RFC's room chat event with the NPC as the speaker: `EVT ROOM CHAT <npc name> <message>`.
- `EVT RESPAWN room=<zone>` is a custom event sent to a player who loses a fight, after they are moved to the respawn zone.
- Talking to a healer while injured returns the healer's heal line followed by `(You feel a warm glow)` in the normal `OK` reply.
- Quest progress and reward messages are extra plain-text lines sent after the command's `OK` reply (see Quest System).
- When a quest step needs key items the player doesn't hold yet, `TALK` replies with that step's `missing_items_dialogue` in the normal `OK` reply.
- When a quest step gives the player key items, their names are appended to the step's dialogue in the same `OK` reply: `OK <dialogue> (You receive Loaf of Bread)`.

### Case-insensitive input
Command names are case-insensitive, as RFC §4.2 requires: `handleTCPConn` uppercases the command once before passing it to `handleLogin` or `commandDispatch`, so `look`, `Look` and `LOOK` all work. The same applies to `CHAT` scopes (`chatDispatcher`), `GROUP` subcommands (`groupFuncDispatcher`) and `MOVE` directions (see Movement directions below). Item and NPC names are matched case-insensitively with `strings.EqualFold`.

### Movement directions
Exits use a fixed set of six directions: `north`, `east`, `south`, `west`, `up` and `down` (the `ZoneMoveDirection` enum in `zones.go`). `parseDirection` converts a direction word to the enum case-insensitively. `MOVE` with any other word, or with a direction the current room has no exit for, returns `ERR 301 NO_EXIT`. `LOOK` still reports exits as plain strings (`"exits": {"north": "chapel"}`), so clients only ever see these six words.

### Item system
Dynamic item system (unique instances, no duplication on TAKE, made available again on DROP, lookup by ID or display name via `strings.EqualFold`) is implemented in `items.go`. Key items are kept outside this system (see Quest System → Key items), so they never conflict with item uniqueness.

[↑ Back to top](#top)

## Combat System

`ATTACK <npc>` against an attackable `enemy` NPC in the player's room starts a **one-on-one, instanced fight**: the NPC is marked occupied (other players get `ERR 405 NPC_CURRENTLY_OCCUPIED`), the room is told the battle has started, and every further `ATTACK` from that player resolves exactly one round. Other players in the room only see the start and end announcements, not the individual rounds.

### Stats
Players start with 100 HP, strength 7, battle skill 7 and dexterity 10. A key item with a `skill_boost` (the Ornate Dagger, +2) raises battle skill when it is received as a quest reward. NPC stats come from `world.yaml` (`hp`, `str`, `battle_skill`, `dex`); the validator requires enemies to have `str` and `battle_skill` of at least 1 and `dex` of at most 90.

### A round
1. Both sides roll a d6 for initiative every round; a tie goes to the player. Initiative only decides who strikes first — both sides still attack.
2. Damage is `strength + roll(1..battle_skill)` (8–14 for a player).
3. The defender dodges completely if a 1–100 roll is not higher than their dexterity (dexterity 10 = 10% dodge chance).
4. If the first attack brings the defender to 0 HP, the second attack doesn't happen.

### Ending a fight
- **Player wins:** the NPC is "defeated" rather than killed — its fight HP was a per-fight copy, so it is immediately available again at full HP. The room is told who won, and any active quest waiting on a `battle` step against that NPC advances.
- **Player loses:** the player respawns in the chapel (`respawnZone`) with half their max HP (rounded up) and receives `EVT RESPAWN room=chapel`. Quests are never failed by a loss; the player can try again.
- **`FLEE`** or a disconnect ends the fight early; the room is told the battle ended prematurely.

### Player status
During a fight the player's status is `engaged`. When a fight ends (or after healing) it is recalculated from HP: `injured` at 30% or less, `weakened` at 60% or less, otherwise `healthy`. `STATUS` reports `{"hp", "max_hp", "status"}`. If the status can't be calculated (max HP of 0 — should never happen, and new players are checked at login), it is set to `unknown` and a warning is logged rather than disconnecting the player.

### Healing
`TALK` to a `healer` NPC while below max HP restores the player to full HP and returns the healer's heal line (from `world.yaml`'s `heal_dialogue`) followed by `(You feel a warm glow)`. At full HP the healer just uses their normal dialogue. The reply deliberately doesn't show the new HP; players check `STATUS`.

### `ATTACK` response
The RFC example shows `attacker_hp`, `target_hp`, `damage` and `status`. This server's reply keeps those and adds the details of the round:

```
C: ATTACK cave bat
S: OK {"attacker_init_roll":4,"target_init_roll":2,"attacker_hp":95,"attacker_max_hp":100,"target_hp":25,"target_max_hp":36,"target_dodged":false,"damage":11,"attacker_dodged":false,"damage_received":5,"status":"engaged"}
```

`status` is the player's own status (`engaged` mid-fight, then `healthy`/`weakened`/`injured` once the fight is over) rather than the RFC example's `"combat"`.

[↑ Back to top](#top)

## Quest System

Quest data model (`quests.go`): a `Quest` (`Name`, `QuestID`, `Description`, `Type`, `Steps []QuestAction`, `Reward *Reward`, `Requires`) has a `QuestType` (`Delivery`, `Fetch`, `Battle`), and each of its steps implements a `QuestAction` interface (`ActionType() string`) as one of `TalkToAction`, `EnterAreaAction`, or `BattleAction`. When `world.yaml` is parsed, the concrete step type is picked based on which of `talk_to`/`enter_area`/`battle` is present on that step (`QuestStep.UnmarshalYAML`, `parser.go`). A `Reward` is optional and can carry one key item, gold, or both. A `talk_to` step can also list key items it gives to the player (`grants_key_items`) or takes from them (`receives_key_items`), plus an optional `missing_items_dialogue`.

Quests are defined under `world.yaml`'s top-level `quests:` section and attached to their quest-giver NPC by ID in that NPC's own `quests:` list — there's no separate global quest registry; a player reaches a quest only through the NPC offering it.

### Quest chains
A quest can optionally name a prerequisite via `requires: <quest_key>` (a bare quest key, `""` if none). `ParseYmlData` validates this at load time by walking each quest's `requires` chain: an unknown prerequisite key is rejected, and so is a chain that loops back on itself, at any length — not just a quest requiring itself directly.

### Per-player progress
Each `Player` has a `Quests map[string]*PlayerQuest` (`quests.go`), tracking `Quest`, `StepIndex`, and `Status` (`Active`/`Completed`) per accepted quest. `NPC.getNPCQuest` picks the first quest that NPC offers which the player doesn't already have, and whose prerequisite (if any) is already `Completed`.

### Commands
`QUEST` (`checkNPCQuest`) reports an NPC's next eligible quest for the calling player without changing any state. `ACCEPT` (`acceptQuest` — a custom addition, see Protocol Implementation) commits the player to that quest, adding it to `player.Quests` as `Active`. `QUESTS` (`printPlayerQuests`) lists everything the calling player has accepted, showing `progress` (`stepIndex/totalSteps`) only while a quest is still `Active`, plus `quest_items` when the current step needs key items (see Protocol Implementation).

### Step progression
Each step type advances inside the handler for the action it describes, only for the calling player, and only when it is that quest's current step:

- `talk_to` — `TALK <npc>` (`getDialogue`, `NPCs.go`): if an active quest's current step targets that NPC, the step's own `dialogue` is sent in place of the NPC's normal lines. If the step has `receives_key_items`, it only advances when the player holds all of them; otherwise the NPC replies with the step's `missing_items_dialogue` (or their normal lines if it has none). When the step advances, its `grants_key_items` are added to the player's key items (announced as `(You receive <name>)` at the end of the dialogue) and its `receives_key_items` are taken away.
- `enter_area` — `MOVE` (`handleMove`, `zones.go`): after the normal `OK room=<zone>` reply, if an active quest's current step names the zone just entered, the step's `message` is sent as an extra plain-text line. Respawning into a zone does not count.
- `battle` — `ATTACK` (`resolveAttackRequest`, `battle.go`): after the final `OK {...}` reply of a fight the player won, the step advances if the defeated NPC is the step's target. A lost fight leaves the step unchanged, so the player can try again.

At most one step advances per action. When the last step completes, `completeQuest` (`quests.go`) marks the quest `Completed` and grants its reward: gold is added to `Player.Gold` and a key item is added to `Player.KeyInventory` (applying its `skill_boost`, if any), each announced with a plain-text line (e.g. `<player> receives 20 gold.`). A quest without a reward sends a short acknowledgement line instead.

### Key items
Key items are quest items, defined under `world.yaml`'s `key_items:` section (`name`, `description`, optional `skill_boost`) and referenced elsewhere by their `key_item.<key>` ID. They are held per player in `Player.KeyInventory`, listed with `KEYITEMS` and described with `EXAMINE KEYITEM <name>`.

They are deliberately kept apart from world items: they never appear in a room, can't be taken or dropped, and every player who completes a quest gets their own copy. The subject's item rules (unique instances, no duplication on `TAKE`) apply to world items only. Key items, like the rest of the quest system, are left to the implementer by RFC §6.1.2.

Two rules decide where a key item can be used:

- **Step grants are temporary.** A key item given by a quest step (`grants_key_items`, e.g. George's loaf) must be taken back by a later step of the same quest (`receives_key_items`, e.g. Kassandra). This keeps quest props from piling up in the player's key items.
- **Rewards are kept.** A key item given as a quest reward stays with the player and can be required by any other quest. *Arms Dealer* uses this: Brannoc takes the Rusty Dagger and the Small Gem, the rewards of the two Kassandra quests, and gives back the Ornate Dagger. It has no `requires`, so it can be accepted early, but it can't be finished until the player holds both items.

Key items, like quest progress, last for the player's connection only.

### Protocol note
The RFC leaves quest progression to the implementer, so the `enter_area` message, the reward lines and the `missing_items_dialogue` reply are additions of this server: plain-text lines sent after the command's normal `OK` reply, not RFC-defined responses.

### Validation
`ValidateWorldData` rejects quest steps whose target NPC or zone doesn't exist, `battle` steps targeting an NPC that isn't an attackable `enemy`, `talk_to` steps targeting a `healer`, `talk_to` steps whose `missing_items_dialogue` is set but blank, and `enter_area` steps with an empty `message`.

Key item references are checked too: every key item named in a step or reward must exist in `key_items:`; a step can't grant the same key item twice; every step grant must be received back later in the same quest (`QUEST_KEY_ITEMS_NOT_BALANCED`); and every received key item must either be granted earlier in the same quest or be granted somewhere in the world. Because the validator only knows which items exist, not which order a player does quests in, *Arms Dealer*'s hand-over is also checked at runtime: the step simply won't advance until the player holds the items.

Example (`world.yaml`) — a two-quest chain, where the second quest only becomes available once the first is completed, and a fetch quest that requires both chain rewards:

```yaml
quests:
  bread_for_kassandra:
    name: A loaf for the needy
    description: George seems to need help with his daily chores.
    type: delivery
    requires: ""
    steps:
      - talk_to: george
        dialogue: "Hey there! Be a champ and take this [keyitem.Loaf] to Kassandra for me, would you? Thanks!"
        grants_key_items:
          - "key_item.bread_loaf"
      - talk_to: kassandra
        dialogue: "What's this, a loaf of bread? So George thinks to woo me with crumbs... ..."
        receives_key_items:
          - "key_item.bread_loaf"
    reward:
      key_item: "key_item.rusty_dagger"

  something_in_the_well:
    name: Something in the Well
    description: Strange sounds rise from the old well at night. Kassandra means to know why.
    type: battle
    requires: bread_for_kassandra
    steps:
      - talk_to: kassandra
        dialogue: "The well sings at night. ..."
      - enter_area: old_well
        message: You hear eerie sounds coming from deep inside the well...
      - battle: cave_bat
      - talk_to: kassandra
        dialogue: "Only a bat... Pity. ..."
    reward:
      key_item: "key_item.small_gem"
      gold: 20

  arms_dealer:
    name: Arms Dealer
    description: Brannoc can mend any blade. He'll also tell you about his arms, at length, whether you ask or not.
    type: fetch
    requires: ""
    steps:
      - talk_to: brannoc
        dialogue: "Come for your daily viewing of these impeccable biceps, have you? ..."
      - talk_to: brannoc
        receives_key_items:
          - "key_item.rusty_dagger"
          - "key_item.small_gem"
        missing_items_dialogue: "Back again without anything to fix? ..."
        dialogue: "Right, stand back. ..."
    reward:
      key_item: "key_item.ornate_dagger"

key_items:
  bread_loaf:
    name: Loaf of Bread
    description: A loaf of bread, apparently. ...

  ornate_dagger:
    name: Ornate Dagger
    description: Beautifully restored, its gem glistens with a familiar warmth. ...
    skill_boost: 2
```

(Dialogue, descriptions and the `key_items:` list shortened here; the full data is in `server/world.yaml`.)

[↑ Back to top](#top)

## World Design

A small, light-hearted fantasy town, defined in `server/world.yaml` and loaded at startup.

### Rooms (10)
New players start in the Taverne; defeated players respawn in the Chapel.

| Room | Exits | NPCs |
|---|---|---|
| The Taverne (`taverne`) | east → Fountain Square | George |
| Fountain Square (`fountain_square`) | north → Chapel, east → Old Well, south → Market Square, west → Taverne | Kassandra |
| Chapel (`chapel`) | east → Churchyard, south → Fountain Square | Father Tieu |
| Churchyard (`churchyard`) | south → Old Well, west → Chapel | Osric |
| Old Well (`old_well`) | north → Churchyard, south → Mill, west → Fountain Square, down → Cistern | — |
| Cistern (`cistern`) | up → Old Well | Cave Bat |
| Market Square (`market_square`) | north → Fountain Square, east → Mill, west → Smithy | Bertha |
| Smithy (`smithy`) | east → Market Square | Brannoc |
| Mill (`mill`) | north → Old Well, east → Fallow Field, west → Market Square | Hilde |
| Fallow Field (`fallow_field`) | west → Mill | Barbarian |

The map has two loops that share the Old Well (Fountain Square → Chapel → Churchyard → Old Well → Fountain Square, and Fountain Square → Market Square → Mill → Old Well → Fountain Square) plus three dead-end branches (Cistern, Smithy, Fallow Field), so a full circuit is possible. No room is empty: each has an NPC or a fixed item.

### NPCs (9), 4 roles
- **Quest givers:** George (taverne keeper), Kassandra (a mysterious figure by the fountain), Bertha (market vendor), Brannoc (smith).
- **Healer:** Father Tieu (chapel priest).
- **General:** Osric (gravedigger), Hilde (miller) — dialogue only; NPCs cycle through their lines on each `TALK`.
- **Enemies:** the Barbarian (hp 90, the tougher fight) and the Cave Bat (hp 36 but hard to hit, dex 30).

### Items (19, 7 obtainable)
Obtainable: Dented Tankard, Empty Coin Purse, Prayer Candle, Wooden Bucket, Metal Shavings, Handful of Wheat Husks, Field Flower. The rest (e.g. Weathered Fountain, Wanted Poster, Anvil, Millstone) are fixed scenery: they show up in `LOOK` but can't be taken. Key items (Loaf of Bread, Rusty Dagger, Small Gem, Ornate Dagger) are quest items held separately from the normal inventory (see Quest System → Key items).

### Quests (4)
- *A loaf for the needy* (George) — deliver bread to Kassandra. Reward: rusty dagger.
- *Something in the Well* (Kassandra, requires the first quest) — find out what is making noise in the old well. Reward: small gem and 20 gold.
- *A Lesson in Manners* (Bertha) — deal with the brute who wrecked her stall. Reward: 10 gold.
- *Arms Dealer* (Brannoc) — bring him the Rusty Dagger and the Small Gem to have the blade restored. Reward: ornate dagger (+2 battle skill).

### Validation at startup
When `world.yaml` is parsed, every exit direction must be one of the six movement directions, and a room can't list the same direction twice (e.g. `north` and `North`); either problem stops the server with a parse error naming the direction and the room.

Then `ValidateWorldData` (`worldDataValidator.go`) checks that every exit, spawn, item and quest reference resolves; every room needs a name and at least one exit; items can only be placed once; NPC roles must match their data (quest givers need quests, enemies must be attackable with valid stats, healers need a heal line and can't be attacked); key items need a name and description and can't have a negative `skill_boost`; no two items or key items may share a display name (compared case-insensitively, so name lookups are never ambiguous); quest key item references must be consistent (see Quest System → Validation); the starting and respawn rooms must exist; and the minimum counts are enforced (≥8 rooms, ≥4 items, ≥2 obtainable, ≥3 NPC roles, ≥2 quests). Any failure stops the server with a list of every problem found.

After `ValidateWorldData` passes, two map checks run before the server starts listening: every room must be reachable from the starting room (breadth-first search, `validateMapConnectivity`), and the map must contain at least one loop (depth-first search, `mapLoopExists`). If either check fails, the server logs the problem and exits. Together with the 8-room minimum, this enforces the subject's requirement of at least 8 interconnected rooms forming a loop.

[↑ Back to top](#top)

## Server Logging

Structured JSON logging via `log/slog` (`slog.NewJSONHandler`, written to stderr), with `INFO`/`WARN`/`ERROR` levels. Every command handler logs its outcome with player name, command, and relevant args/state; connects, disconnects (graceful, idle-timeout, and error paths), and abuse events (spam timeouts, softbans, connection-flood detection) are all logged with the remote address. `PLAYER_CLEANUP_COMPLETE` and per-zone leave events are logged on disconnect/cleanup. JSON output plus slog's built-in timestamps satisfy the "structured, timestamped, parseable" logging requirement.

[↑ Back to top](#top)

## Group Contributions

The Answer Protocol is designed as a group project for 2–3 learners. As no eligible peers were available to form a group, I completed the entire project on my own.

- **Server** (Go): protocol, world loading and validation, combat, quests
- **CLI clients**: Go and Rust versions
- **World design**: rooms, NPCs, items, dialogue and quests
- **Documentation**: this README
- **GUI client**: still in progress

[↑ Back to top](#top)

## Building and Running

Requirements: Go (the server's `go.mod` targets Go 1.27; single dependency `gopkg.in/yaml.v3`) and Rust/Cargo for the Rust CLI (dependencies `anyhow`, `ctrlc`).

From the repository root:

| Target | What it does |
|---|---|
| `make install` | Downloads Go modules and builds the server and both CLI clients |
| `make run-server` | Starts the server on TCP port 4242 (reads `server/world.yaml`) |
| `make run-client` | Starts the Rust CLI client |
| `make run-go-client` | Starts the Go CLI client |
| `make lint` | `go vet` on the server and Go CLI, `cargo check` on the Rust CLI |
| `make clean` | `go clean` / `cargo clean` |

`make -C server format` and `make -C CLI format` run `gofmt` (and `cargo fmt` for the CLI). `build-GUI` and `run-client-gui` exist as placeholders until the GUI client is written.

Both CLI clients connect to `localhost:4242`, prompt for a username, and send `CONNECT <username>` automatically. After that, type protocol commands directly (e.g. `LOOK`, `MOVE east`, `TALK george`). `QUIT`, Ctrl+C or Ctrl+D disconnects.

[↑ Back to top](#top)

## Testing

NPCs are addressed by their display name, case-insensitively (e.g. `TALK george`, `ATTACK cave bat`), or by their full ID (`npc.cave_bat`).

### Multiplayer
Start the server, then open several terminals and run a CLI client in each with a different username. Check that `WHO`, `CHAT ROOM`/`CHAT GLOBAL`, room enter/leave events, groups (`GROUP CREATE`/`INVITE`/`JOIN`/`LEAVE`), and item uniqueness (one player `TAKE`s an item, the other no longer sees it in `LOOK`; `DROP` makes it available again) all behave as expected.

### Combat
Go to the Fallow Field (taverne → east → south → east → east) and `ATTACK barbarian` repeatedly until the fight ends. While fighting, try `MOVE` (refused with `666`), and from a second client try to `ATTACK` the same NPC (refused with `405`). Lose a fight to check the respawn in the chapel at half HP, then `TALK father tieu` to heal and check `STATUS`. Try `FLEE` mid-fight.

### Quests
A full walkthrough of the quest chain:
1. In the Taverne: `QUEST george`, `ACCEPT george`, `TALK george` → the reply ends with `(You receive Loaf of Bread)`. `KEYITEMS` now lists the loaf; `EXAMINE KEYITEM loaf of bread` describes it.
2. Fountain Square: `TALK kassandra` → the loaf is handed over, quest complete, reward line shown. `QUESTS` shows it as completed.
3. `QUEST kassandra` now offers *Something in the Well*; `ACCEPT kassandra`, `TALK kassandra`.
4. `MOVE east` into the Old Well → the eerie-sounds message appears.
5. `MOVE down`, `ATTACK cave bat` until it is defeated.
6. Back to Fountain Square, `TALK kassandra` → quest complete; `GOLD` shows 20.
7. In the Smithy: `ACCEPT brannoc`, `TALK brannoc` (hint), then `TALK brannoc` again → the Rusty Dagger and Small Gem are handed over, the Ornate Dagger is received with `+2 BattleSkill`.

*Arms Dealer* can also be accepted early: before both items are held, `TALK brannoc` gives his missing-items line and `QUESTS` shows `"quest_items":"1/2"` (or `0/2`).

*A Lesson in Manners* works the same way with Bertha in the Market Square and the Barbarian in the Fallow Field.

### EXAMINE
`EXAMINE ITEM tankard` in the Taverne (also after `TAKE tankard`), `EXAMINE NPC george`, `EXAMINE KEYITEM rusty dagger` before and after handing it to Brannoc (`ITEM_NOT_IN_INVENTORY` once it's gone), and `EXAMINE FOO x` (`INVALID_ARGS`).

### World validation
Break `world.yaml` on purpose (an exit to a missing room, an exit with an unknown direction such as `northwest`, a quest step targeting a missing NPC, a healer without `heal_dialogue`, a key item with the same name as an item, a `receives_key_items` entry nobody grants, a step grant that is never received back) and check the server refuses to start and lists each problem.

[↑ Back to top](#top)

## Resources

- RFC 42TAP (attached protocol spec) — primary reference for commands, events, and error codes.
- AI usage (Claude), used during development for:
  - Running `golangci-lint` against the server, since it couldn't be installed locally
  - Identifying a softban bypass bug (ban keys included the client's ephemeral port)
  - Designing the quest data model and response shapes, and splitting `ACCEPT` out of `QUEST`
  - Spotting concurrency bugs: a cross-zone mutex-ordering deadlock risk, a self-deadlock when a player disconnects mid-fight, and lock/unlock pairing in the quest-step hooks
  - Spotting a redundant-recursion bug in the map cycle-detection helper
  - Suggesting world-building ideas (room layout, NPCs, items, draft descriptions and dialogue), which were then edited and chosen by hand
  - Simulating fights to help balance enemy stats
  - Talking through the key item design (separating key items from world items, the step-grant rule) and its validation rules, and reviewing the `EXAMINE`, `KEYITEMS` and `quest_items` code
  - Reviewing code changes (e.g. error wrapping in the login path) and checking `world.yaml` for broken references
  - Drafting and formatting this README

[↑ Back to top](#top)
