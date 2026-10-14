# TAP GUI — HUD Design Record

*42 school project "The Answer Protocol" (TAP): design decisions for the GUI client's game screen. Last updated 2026-10-14.*

Legend: **[R]** required by the subject PDF or RFC 42TAP · **[O]** optional, not mentioned in the subject but allowed · **[ext]** uses a server extension (a command not in the RFC) · **TBD** not decided yet.

Which of these elements are built so far is listed in the README (GUI Client section).

---

## 1. Basics

- Go with Fyne v2.8.1. Files in `GUI/`:
  - `gui.go`: entry point, window and channels
  - `loginScreen.go`, `gameScreen.go`, `overlay.go`, `chatWindows.go`, `buttons.go`, `zoneImages.go`: screens and widgets
  - `commandRouter.go`, `replyHandlers.go`, `battleScreen.go`: the router goroutine and its handlers
  - `tapConnection.go`: the TCP connection
  - `TAPResponsesParser.go`: turns each server line into a typed reply
  - `utils.go`: label helpers
- Fixed window of 1280 × 900, designed for a 1920 × 1080 screen. The zone background image is 640 × 360 pixel art shown at 2× (1280 × 720); the rest of the height holds the rows above and below it.
- The login window is 400 × 200.
- Many widgets sit on top of the background image. A show/hide button lets players see the artwork (section 10).
- All choices follow the subject PDF (V.4 GUI Client) and RFC 42TAP. The PDF requires CLI and GUI clients to be interchangeable between groups, so the GUI accepts both the RFC and the PDF forms of a reply where they differ, and server extensions have a fallback (section 9).

---

## 2. Layout diagram

Positions follow the original sketch plus the revisions in section 15 and will be adjusted while building. Numbers match the element table in section 3. The diagram uses plain ASCII so it lines up in any font.

```
+----------------------------------------------------------------------------+
| [1] HP [######----]            [2] ROOM NAME                [3] Online: 12 |
+----------------------------------------------------------------------------+
| [4] Status: healthy         [6] [Room details]                [8] Room: 2  |
| [5] Gold: 40                +----------------+              +------------+ |
|                             | [7] zone info  |              | [9] D-PAD  | |
|   BG: zone background       | items / NPCs   |              |   N   up   | |
|   1280 x 720, 2x art        | exits / ...    |              | W   E      | |
|                             | (fight: [22]   |              |   S   down | |
|                             |  battle log)   |              +------------+ |
|                             +----------------+              +------------+ |
|                                                             | [10] KEY   | |
|                                                             | ITEMS      | |
|                                                             +------------+ |
| [15] error pop-up (fades after 3 s)                         +------------+ |
| +--------------+ +--------------+  [12]                     | [11] INV.  | |
| | [13] CHAT    | | [14] LOG     |  [LOOK]                   | herbs      | |
| | Global (a)>> | | You open ... |  [WHO]                    | tankard    | |
| +--------------+ +--------------+  [STATUS]                 +------------+ |
| [16] [ manual command ... ]        [QUESTS]                                |
+----------------------------------------------------------------------------+
| [17] RPG MENU                                         [20] [Show/Hide UI]  |
|  > Talk > Attack > Quest > Group > Chat > Take > Drop   [18] > Flee        |
|                                                               [21] [Quit]  |
+----------------------------------------------------------------------------+
       [19] typed-input dialogue pops up when a menu entry needs text
```

---

## 3. Elements

| # | Element | What it does | Req |
|---|---|---|---|
| 1 | HP bar | Text row, left. Current/max HP. Updated from STATUS replies and from every ATTACK reply. | [O] |
| 2 | Room name | Text row, centre. Current room name from LOOK. | [R] (room details) |
| 3 | `Online: #` | Text row, hard right. Players on the server, from `EVT STATS players=` (sent after login and on every login/logout). A WHO reply also updates it. | [R] counter |
| 4 | Status label | On the image, top-left. The server's own status string from STATUS (e.g. healthy / weakened / injured / engaged). | [O] |
| 5 | Gold | On the image, under the status. Gold amount from the GOLD command. | [O][ext] |
| 6 | Room details button | Centred. Shows/hides the zone info box (7). Disabled while the battle log (22) is showing. | [O] |
| 7 | Zone info box | Below 6, centred. A description row on top, then four columns: Players, NPCs, Exits, Items. Each column is a list that scrolls when it holds more than 4 entries. Exit rows read `Direction: Destination` (e.g. `East: Fountain Square`). **Updates live whether showing or not.** Shown at start, hideable with 6. | [R] |
| 8 | `Room: #` | On the image, hard right. Players in the current room: set from LOOK's players list and adjusted by presence events (section 12). | [R] counter |
| 9 | D-pad | Right side, below 8. MOVE buttons (north, south, east, west, up, down). Directions not in the LOOK exits are greyed out. Disabled during room transitions until the LOOK reply arrives, and during a rate-limit lockout (ERR 750). | [R] MOVE |
| 10 | Key items | Right side, above the inventory. Same style as the inventory (label plus a list that grows and shrinks). Shows "Feature not available" if the server doesn't support it. | [O][ext] |
| 11 | Inventory | The "Inventory" label is always visible. The list under it grows and shrinks as items are taken and dropped; an empty inventory shows only the label and one blank row. At most 4 rows are shown, then a scroll bar appears. | [R] |
| 12 | Info buttons | LOOK, WHO, STATUS, QUESTS: small buttons stacked vertically, left of the inventory. Results are formatted log messages (section 5). | [R] buttons |
| 13 | Chat window | Bottom left. Chat messages (including the player's own), NPC dialogue and NPC fight announcements (section 4). | [R] |
| 14 | Log window | Next to the chat window. Everything else (section 4). | [R] |
| 15 | Error pop-up | Above the chat window. Shows an error, then fades after 3 seconds. Errors are also written to the log. | [O] |
| 16 | Manual command entry | Narrow field under the chat windows (section 11). | [O] |
| 17 | RPG menu | Early-Final-Fantasy-style menu below the image (section 8). | [R] buttons |
| 18 | Flee entry | Appears in the RPG menu only during a battle; disappears when the battle ends (win, flee or defeat). | [O][ext] |
| 19 | Typed-input dialogue | Pop-up for text input (chat message, group invite/join name). | [O] |
| 20 | Show/Hide UI | Top right of the RPG menu section (section 10). | [O] |
| 21 | Quit | Small standalone button at the bottom right of the RPG menu section. Uses the existing quit path (`QuitRequest`, `handleQuit`). | [R] QUIT |
| 22 | Battle log | Takes the place of the zone info box (7) during a fight and shows one entry per round (section 7). | [O] |

---

## 4. Chat window and log window

- Two separate windows, no tabs. The PDF requires the chat view (Global / Room / Group) to be separate from the log view.
- Both windows scroll so players can read the history. New messages scroll into view **only when the player is already at the bottom**. Each window keeps the last 250 lines.

**Chat window**

| Source | Shown as |
|---|---|
| Another player's chat (`EVT <scope> CHAT`) | `Global (pname) >> message` (scope: Global, Room or Group) |
| The player's own chat | `Global (YOU) << message`, added by the client as soon as it is sent |
| NPC fight announcement (`EVT ROOM CHAT <<npc.id>> …`) | `Room (<<Cave Bat>>) >> …`: the NPC ID inside `<<…>>` is turned into a label (section 8) |
| NPC dialogue (TALK reply) | `NPC (George): dialogue` (section 6) |

- The server does not send a chat message back to its sender, so the client adds the player's own line itself. This gives the immediate feedback players know from online RPGs. If the server rejects the message (for example not in a group, rate limit), the error is shown as for any other command.

**Log window**

- Everything else: quest and step messages, errors, and the formatted results of the four info buttons.
- Errors also appear in the 3-second pop-up (15).

---

## 5. Info buttons (LOOK, WHO, STATUS, QUESTS)

- Each press sends a fresh command. The result is written to the log as a formatted message, with item and NPC names derived from their IDs (section 8). There are no pop-up windows for them.
- Every command can also be typed in the manual entry (section 11), and typed commands are handled the same way as the buttons. WHO and QUESTS are never sent automatically, so their replies are always logged. Refreshing the quest list is up to the player.
- The GUI also sends LOOK and STATUS automatically to keep the display current, and replies don't name the command they answer. A LOOK or STATUS sent from a button (or typed) therefore records the time of the send for that command type:
  - a matching reply within 3.5 seconds is logged and clears the record;
  - a reply outside that window is treated as automatic and only updates the display;
  - an old record stops counting by itself, so a rejected query (ERR 747) cannot leave a mark set.
- An automatic LOOK still updates the zone info box and the counters; it just doesn't print to the log.

---

## 6. NPC dialogue

- RFC 42TAP shows the TALK reply as plain text (`OK <dialogue>`); the subject PDF shows JSON (`OK { "npc": …, "dialogue": … }`). This project's server sends the JSON form, and the GUI accepts both:
  - JSON with a non-empty `dialogue`: the text and the NPC come from the reply.
  - Anything else after `OK `: the whole text is the dialogue, and the NPC is the target of the player's last TALK command.
- The line goes to the chat window as `NPC (<name>): <dialogue>`, with the name turned into a label (section 8). If no name is known, it reads `NPC (NPC)`.
- A dialogue that contains `(You feel a warm glow)` (this server's healer suffix) makes the GUI send STATUS, so the HP bar updates after healing. On other servers the suffix never appears, and the player can press STATUS.

---

## 7. Battle log (22)

- **Placement:** shares the zone info box's space; only one of the two is visible at a time. While it is showing, the Room details button (6) is disabled. When the fight ends, the zone info box comes back only if it was showing before the fight.
- **Content:** one entry per ATTACK reply. The first line names both initiative rolls and who goes first; the second line gives the hits and dodges in initiative order, with the enemy's HP after the player's hit. A side that was defeated before its turn is left out.
- **Turns on** when an ATTACK reply arrives.
- **Turns off** when any of these happens:
  - an ATTACK reply with the player's or the enemy's HP at 0;
  - `OK battle ended` (FLEE);
  - `EVT RESPAWN`;
  - `OK room=` (a successful move);
  - a dropped connection or quitting;
  - 10 seconds without an ATTACK reply. This is a safety net for servers whose replies the GUI can't fully read, so the box can't stay hidden.
- The fight is detected from HP values rather than the `status` string, since status words differ between servers.

---

## 8. RPG menu and labels

Main entries: **Talk, Attack, Quest, Group, Chat, Take, Drop**, plus **Flee** during a battle only.

- **Talk / Attack / Quest:** a submenu of the NPCs in the room (from LOOK's `npcs` list). Clicking one sends the command.
- **Take:** submenu of the items in the room. **Drop:** submenu of the inventory. Clicking one sends the command.
- **Group:** expands into CREATE, INVITE, JOIN, LEAVE. INVITE and JOIN always use typed input.
- **Chat:** pick a scope (Global / Room / Group), then type the message in the dialogue box. Group chat outside a group simply shows the server's error (ERR 401 NOT_IN_GROUP).
- **IDs and names:** menu entries send the item/NPC ID. The server accepts either IDs or display names, and so does the manual entry.
- **Labels:** made from IDs on the client only, with no extra requests. The exact prefixes `item.`, `key_item.`, `npc.`, `zone.`, `room.` and `loc.` are removed (a `.` elsewhere in a name is kept, since other servers may use one), and the rest is converted from snake case to words: `npc.father_tieu` becomes `Father Tieu`, `zone.fountain_square` becomes `Fountain Square`. Labels behave the same on every server.
- **Room changes:** the menu is unavailable after a MOVE until the LOOK reply arrives. It always unlocks again on an error or a dropped connection.
- **Flee** uses the server's FLEE command. During a battle the server rejects MOVE, TAKE, DROP and QUEST.
- **Rate-limit lockout (ERR 750):** every widget that sends a non-query command is disabled for the lockout, including the manual entry, because commands sent during a timeout count as strikes towards a longer ban.

---

## 9. Server extensions and PDF V.4 coverage

Extensions not in the RFC: GOLD, KEYITEMS, EXAMINE, FLEE and the JSON reply of WHO. If a server doesn't support one, its widget shows **"Feature not available"**. Servers from other groups may not support them; that is the only case this covers.

| PDF V.4 requirement | Where |
|---|---|
| Room details, items, NPCs, exits with real-time updates | 7 (live) |
| Player inventory plus TAKE / DROP buttons | 11, RPG menu |
| Item IDs and display names handled | server resolves both; manual entry 16 |
| Room view updated after TAKE / DROP | LOOK sent after the reply |
| Chat view (Global / Room / Group) separate from the log | 13 and 14 |
| Buttons for LOOK, MOVE, TAKE, DROP, TALK, ATTACK, STATUS, QUEST, QUESTS, WHO, GROUP, QUIT | 12, 9, 17, 21 |
| Counters for players in the room and on the server | 8 and 3 |
| NPC interaction with TALK and dialogue display | RPG menu, chat window |

---

## 10. Show/Hide UI (20)

- Hides the HUD overlays so the artwork is visible, and shows them again exactly as they were, **remembering each element's own state**. Hiding everything except the button itself is acceptable.

---

## 11. Manual command entry (16)

- A permanent, narrow field under the chat windows for players who prefer typing commands (also useful for testing).
- Typed commands are handled like button commands: the same room-change lock, flee tracking, LOOK/STATUS mark and lockout.
- A typed `QUIT` closes the program, like the Quit button.
- A typed `CHAT <scope> <text>` also adds the player's own line to the chat window (section 4), the same way as a message sent from the chat menu.
- The targets of typed `ATTACK` and `TALK` commands are remembered for the battle log and for naming the NPC in plain-text dialogue.

---

## 12. Data refresh plan

- The `Room: #` counter (8) is set from each LOOK reply and adjusted by one for every `EVT ROOM PRESENCE ENTER` (+1) or `LEAVE` (−1), so it changes without sending a query.
- LOOK after `OK room=`, `EVT RESPAWN`, and the player's own `OK taken=` / `OK dropped=`. LOOK is also polled while another player is in the room, because the protocol has no item events (interval TBD, section 14).
- After login: LOOK, STATUS and GOLD.
- Server player count comes from `EVT STATS`; WHO is never sent automatically.
- INVENTORY after the player's own TAKE/DROP. STATUS after respawn and healer TALK. GOLD/KEYITEMS after TALK, QUEST, ATTACK and entering a zone.
- All automatic read-only queries go through the server's query bucket.

---

## 13. Server-side changes made for the GUI

- The query bucket (LOOK, WHO, STATUS, GOLD) was loosened to capacity 10, refill 1.2 per second, so a real player is unlikely to hit ERR 747.
- Login names that could imitate a server line or ID are rejected: reserved words such as `OK`, `ERR` and `EVT`, names containing a space, and names starting with `<`, `{`, `[`, `(`, `npc.`, `item.`, `zone.` or `key_item.`.
- TALK replies use the subject PDF's JSON form, and LOOK exits carry full room IDs, so the GUI can label them the same way as items and NPCs.

---

## 14. Open design points

- LOOK poll interval and its start/stop conditions (section 12).

---

## 15. Design revisions

**2026-10-10**, changes to the first sketch:

- Status and gold moved onto the image (top-left, gold under status) so the text row above the image stays one row: HP bar, room name, Online.
- Room details button centred, with the zone info box centred below it.
- Online and Room right aligned; the d-pad and the lower right widgets right aligned too.
- Error pop-up placed above the chat window.
- Chat and log are two separate windows (chat left, log next to it) instead of tabs.
- The four info buttons are stacked vertically as small buttons, left of the inventory. The planned status pop-up was dropped; results go to the log.
- Manual command entry is narrow and sits directly under the chat windows.
- Quit is a small standalone button at the bottom right of the RPG menu section.
- Inventory: label always visible, list grows and shrinks, at most 4 rows then a scroll bar.
- Key items use the inventory's style. Unsupported extension widgets show "Feature not available".
- Group chat outside a group shows the server's error.

**2026-10-12**

- NPC and item labels come from the LOOK IDs (prefix removed, snake case converted) instead of EXAMINE lookups with a cache.
- The room counter is adjusted by presence events instead of re-sending LOOK.
- The player's own chat message is added to the chat window when sent, so there is a visible confirmation that it went out.

**2026-10-13**

- NPC dialogue goes to the chat window instead of the log, and both TALK reply forms are accepted.
- A battle log replaces the zone info box during a fight; the Room details button is disabled meanwhile.
- Exit labels are made from full room IDs like other labels.
- The layout diagram was redrawn in plain ASCII so it lines up in every font.
