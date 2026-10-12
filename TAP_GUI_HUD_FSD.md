# TAP GUI — HUD Design Record

*42 school group project "The Answer Protocol" (TAP): design decisions for the GUI client's game screen. Last updated 2026-10-12.*

Legend: **[R]** required by the subject PDF or RFC 42TAP · **[O]** optional, not mentioned in the subject but allowed · **[ext]** uses a server extension (a command not in the RFC) · **TBD** not decided yet.

---

## 1. Basics

- Go with Fyne v2.8.1. The game screen is built on the existing login/connection code (`gui.go`, `gameScreen.go`, `commandRouter.go`, `replyHandlers.go`, `tapConnection.go`, `TAPResponsesParser.go`, `zoneImages.go`, `buttons.go`).
- Fixed window: width 1280, height about 950 (taller rather than wider), designed for a 1920×1080 screen. The zone background image is 640×360 pixel art shown at 2× (1280×720).
- The login window stays 400×200.
- Many widgets sit on top of the background image. A show/hide button lets players see the artwork (section 8).
- All choices follow the subject PDF (V.4 GUI Client) and RFC 42TAP. The PDF requires CLI and GUI clients to be interchangeable between groups, which is why server extensions have a fallback (section 7).

---

## 2. Layout diagram

Positions come from the original sketch plus the revisions in section 13, and will be adjusted while building. Numbers match the element table in section 3.

```
┌────────────────────────────────────────────────────────────────────────────┐
│ ① HP [██████░░░░]              ② ROOM NAME                    ③ Online: 12 │
├────────────────────────────────────────────────────────────────────────────┤
│ ④ Status: healthy            ⑥ [Room details]                    ⑧ Room: 2 │
│ ⑤ Gold: 40                   ┌──────────────┐               ┌────────────┐ │
│                              │ ⑦ zone info  │               │  ⑨ D-PAD   │ │
│   BG: zone background        │ items / NPCs │               │ N  W E  S  │ │
│   1280 x 720, 2x pixel art   │ exits / ...  │               │ up    down │ │
│                              └──────────────┘               └────────────┘ │
│                                                             ┌────────────┐ │
│                                                             │ ⑩ KEY ITEMS│ │
│                                                             │ rusty key  │ │
│                                                             └────────────┘ │
│                                                             ┌────────────┐ │
│ ⑮ error pop-up (fades after 3 s)                            │ ⑪ INVENTORY│ │
│ ┌─────────────┐ ┌─────────────┐  ⑫                          │ herbs      │ │
│ │ ⑬ CHAT      │ │ ⑭ LOG       │  [LOOK]                     │ tankard    │ │
│ │ GLOBAL (a)>>│ │ You open the│  [WHO]                      └────────────┘ │
│ └─────────────┘ └─────────────┘  [STATUS]                                  │
│ ⑯ [ manual command ...  ]        [QUESTS]                                  │
├────────────────────────────────────────────────────────────────────────────┤
│ ⑰ RPG MENU                                                ⑳ [Show/Hide UI] │
│  ▸Talk ▸Attack ▸Quest ▸Group ▸Chat ▸Take ▸Drop  ⑱ ▸Flee (battle only)      │
│                                                                   ㉑ [Quit] │
└────────────────────────────────────────────────────────────────────────────┘
          ⑲ typed-input dialogue pops up when a menu entry needs text
```

---

## 3. Elements

| # | Element | What it does | Req |
|---|---|---|---|
| ① | HP bar | Text row, left. Current/max HP. Updated from STATUS, ATTACK replies, and refreshes after healer TALK, fights and respawn. | [O] |
| ② | Room name | Text row, centre. Current room name from LOOK. | [R] (room details) |
| ③ | `Online: #` | Text row, hard right. Players on the server, from `EVT STATS players=` (sent after login and on every login/logout). | [R] counter |
| ④ | Status label | On the image, top-left. Player status (healthy / weakened / injured / engaged / unknown). Displays the server's own status string. Reduces the need to press STATUS. | [O] |
| ⑤ | Gold | On the image, under the status. Gold amount (GOLD command). | [O][ext] |
| ⑥ | Room details button | Centred. Shows/hides the zone info box ⑦. | [O] |
| ⑦ | Zone info box | Below ⑥, centred. A description row on top, then four columns: Players, NPCs, Exits, Items. Each column is a list that shows a scroll bar when it holds more than 4 entries. Exit rows read `Direction: Destination`, with the direction capitalised and the destination's snake case converted to words (e.g. `East: Fountain Square`). **Updates live whether showing or not.** Shown at start, hideable with ⑥. | [R] |
| ⑧ | `Room: #` | On the image, hard right. Players in the current room: set from LOOK's players list and adjusted by presence events (section 10). | [R] counter |
| ⑨ | D-pad | Right side, below ⑧. MOVE buttons (north, south, east, west, up, down). Directions not in the LOOK exits list are greyed out. Disabled during room transitions until the LOOK reply arrives, and during a rate-limit lockout (ERR 750). | [R] MOVE |
| ⑩ | Key items | Right side, above the inventory. Same visual style as the inventory (label plus a list that grows and shrinks). Shows "Feature not available" if the server doesn't support it. | [O][ext] |
| ⑪ | Inventory | The "Inventory" label is always visible. The list under it grows and shrinks as items are taken and dropped; an empty inventory shows only the label and one blank row. At most 4 rows are shown, then a scroll bar appears. | [R] |
| ⑫ | Info buttons | LOOK, WHO, STATUS, QUESTS: small buttons stacked vertically, left of the inventory. Results are formatted log messages (section 5). | [R] buttons |
| ⑬ | Chat window | Bottom left. Other players' chat and NPC fight announcements (section 4). | [R] |
| ⑭ | Log window | Next to the chat window. Everything else (section 4). | [R] |
| ⑮ | Error pop-up | Just above the chat window. Shows an error, then fades after 3 seconds. Errors are also written to the log. | [O] |
| ⑯ | Manual command entry | Narrow field under the chat windows (section 9). | [O] |
| ⑰ | RPG menu | Early-Final-Fantasy-style menu below the image (section 6). | [R] buttons |
| ⑱ | Flee entry | Appears in the RPG menu only during a battle; disappears when the battle ends (win, flee or death). | [O][ext] |
| ⑲ | Typed-input dialogue | Pop-up for text input (chat message, group invite/join name). | [O] |
| ⑳ | Show/Hide UI | Top right of the RPG menu section (section 8). | [O] |
| ㉑ | Quit | Small standalone button at the bottom right of the RPG menu section. Uses the existing quit path (`quitRequest`, `handleQuit`). | [R] QUIT |

---

## 4. Chat window and log window

- Two separate windows, no tabs. The PDF requires the chat view (Global / Room / Group) to be separate from the log view.
- **Chat** lines carry a scope prefix, for example `GLOBAL (pname) >> message`. NPC fight announcements arrive as `EVT ROOM CHAT` events from a sender like `<<npc.name>>` and appear here too.
- Players see only **other** players' chat messages (the server doesn't send a message back to its sender).
- The **log** holds everything else: TALK dialogue, ATTACK results, quest and step messages, errors, and the formatted results of the four info buttons.
- Both windows scroll so players can read the history. New messages scroll into view **only when the player is already at the bottom**.
- Errors go to the log, plus the 3-second pop-up (⑮).

---

## 5. Info buttons (LOOK, WHO, STATUS, QUESTS)

- Each press sends a fresh command. The result is written to the log as a user-friendly formatted message, with item and NPC names derived from their IDs (see section 6). There are no popup windows for them.
- Every command can also be typed in the manual entry (section 9), and typed commands are handled the same way as the buttons. WHO and QUESTS are never sent automatically (a button press or typed), so their replies are always logged. Refreshing the quest list is up to the player.
- The GUI also sends LOOK and STATUS automatically to keep the display current, and replies don't name the command they answer. A LOOK/STATUS sent from a button (or typed) therefore records the time of the send for that command type. A matching reply that arrives within the expiry window is logged and clears the record; a reply outside the window is treated as automatic and only updates the display. Because an old record stops counting by itself, a rejected query (ERR 747) cannot leave a mark set. Expiry window: 3.5 seconds.
- An automatic LOOK still updates the zone info box and the counters; it just doesn't print to the log.

---

## 6. RPG menu

Main entries: **Talk, Attack, Quest, Group, Chat, Take, Drop**, plus **Flee** during a battle only.

- **Talk / Attack / Quest:** a submenu of the NPCs in the room (from LOOK's `npcs` list). Clicking one sends the command.
- **Take:** submenu of the items in the room. **Drop:** submenu of the inventory. Clicking one sends the command.
- **Group:** expands into CREATE, INVITE, JOIN, LEAVE. INVITE and JOIN always use typed input.
- **Chat:** pick a scope (Global / Room / Group), then type the message in the dialogue box. Group chat outside a group simply shows the server's error (ERR 401 NOT_IN_GROUP).
- Menu entries send the item/NPC ID. The server accepts either IDs or display names, and the manual entry accepts both as well.
- **Entry labels:** derived from the IDs that LOOK returns, on the client only. The exact prefixes `item.`, `npc.` and `keyitem.` are removed (a `.` elsewhere in a name is kept, since other servers may use one), and the rest is converted from snake case to words. There are no EXAMINE requests and no cache, so labels behave the same on every server.
- **Room changes:** the menu is unavailable after a MOVE until the LOOK reply arrives. It always unlocks again on an error or a dropped connection.
- **Flee** uses the server's FLEE command. During a battle the server rejects MOVE, TAKE, DROP and QUEST.
- **Rate-limit lockout (ERR 750):** every widget that sends a non-query command is disabled for the lockout, including the manual entry, because commands sent during a timeout count as strikes towards a longer ban.

---

## 7. Server extensions and PDF V.4 coverage

Extensions not in the RFC: GOLD, KEYITEMS, EXAMINE, FLEE and the JSON reply of WHO. If a server doesn't support one, its widget shows **"Feature not available"**. Servers from other groups may not support them; that is the only case this covers.

| PDF V.4 requirement | Where |
|---|---|
| Room details, items, NPCs, exits with real-time updates | ⑦ (live) |
| Player inventory plus TAKE / DROP buttons | ⑪, RPG menu |
| Item IDs and display names handled | server resolves both; manual entry ⑯ |
| Room view updated after TAKE / DROP | LOOK sent after the reply |
| Chat view (Global / Room / Group) separate from the log | ⑬ and ⑭ |
| Buttons for LOOK, MOVE, TAKE, DROP, TALK, ATTACK, STATUS, QUEST, QUESTS, WHO, GROUP, QUIT | ⑫, ⑨, ⑰, ㉑ |
| Counters for players in the room and on the server | ⑧ and ③ |
| NPC interaction with TALK and dialogue display | RPG menu, log |

---

## 8. Show/Hide UI (⑳)

- Hides the HUD overlays so the artwork is visible, and shows them again exactly as they were, **remembering each element's own state**. Hiding everything except the button itself is acceptable.

---

## 9. Manual command entry (⑯)

- A permanent, narrow field under the chat windows for players who prefer typing commands (also useful for testing).
- Typed commands are handled like button commands: the same room-change lock, flee tracking, LOOK/STATUS mark and lockout. A typed `QUIT` uses the same graceful quit path as the Quit button.

---

## 10. Data refresh plan

- The `Room: #` counter (⑧) is set from each LOOK reply and adjusted by one for every `EVT ROOM PRESENCE ENTER` (+1) or `LEAVE` (−1), so it changes without sending a query.
- LOOK after `OK room=`, `EVT RESPAWN`, and the player's own `OK taken=` / `OK dropped=`. LOOK is also polled while another player is in the room, because the protocol has no item events. Poll interval: 3–5 s discussed, exact value not decided.
- Server player count comes from `EVT STATS`; WHO is never sent automatically.
- INVENTORY after the player's own TAKE/DROP. STATUS after fights, healer TALK and respawn. GOLD/KEYITEMS after TALK, QUEST, ATTACK and entering a zone.
- All automatic read-only queries go through the server's query bucket.

---

## 11. Planned server-side changes

- Loosen the query bucket (LOOK, WHO, STATUS, GOLD) for the GUI's requests so real players are unlikely to hit ERR 747. Currently: capacity 8, refill 1 per second.
- Login-name blacklist (names that imitate server replies).

---

## 12. Open design points

- Expiry length and implementation of the LOOK/STATUS mark (section 5).
- LOOK poll interval and its start/stop conditions (section 10).

---

## 13. Design revisions

Changes made to the first sketch, in the order they were decided (2026-10-10):

- Status and gold moved onto the image (top-left, gold under status) so the text row above the image stays one row: HP bar, room name, Online.
- Room details button centred, with the zone info box centred below it.
- Online and Room hard right aligned; the d-pad and lower right widgets right aligned too.
- Error pop-up placed just above the chat window.
- Chat and log are two separate windows (chat left, log middle) instead of tabs.
- The four info buttons are stacked vertically as small buttons, left of the inventory. The planned status popup was dropped; results go to the log.
- Manual command entry is narrow and sits directly under the chat windows.
- Quit is a small standalone button at the bottom right of the RPG menu section.
- Inventory: label always visible, list grows and shrinks, at most 4 rows then a scroll bar.
- Key items use the inventory's style. Unsupported extension widgets show "Feature not available".
- Group chat outside a group shows the server's error.

Revision (2026-10-12):

- NPC and item labels come from the LOOK IDs (prefix removed, snake case converted) instead of EXAMINE lookups with a cache; the EXAMINE rate-limit bucket was removed from the server.
- The room counter is adjusted by presence events instead of re-sending LOOK.