package main

import (
	"fmt"
	"log/slog"
	"sync"
)

// parties holds all active groups, keyed by group ID.
var (
	parties   = make(map[string]*Group)
	partiesMu sync.Mutex
)

// Group represents a party of players sharing chat and invites.
type Group struct {
	GroupID    string
	LeaderName string
	Members    map[string]*Player
	GroupMu    sync.Mutex
}

// groupFuncDispatcher routes a GROUP subcommand (CREATE, INVITE, JOIN, LEAVE).
func groupFuncDispatcher(subcommand, subargs string, player *Player) {
	pname := player.getPlayerName()
	switch subcommand {
	case "CREATE":
		group, err := GroupCreate(player)
		if err != nil {
			_, _ = fmt.Fprintln(player.Conn, err.Error())
			slog.Info(err.Error(), "player", pname, "command", subcommand, "subargs", subargs)
			return
		}

		_, _ = fmt.Fprintln(player.Conn, "OK group="+group.GroupID)
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK group="+group.GroupID, "command", "GROUP", "subcommand", subcommand)

	case "INVITE":
		onlinePlayersMu.Lock()
		invitee, ok := onlinePlayers[subargs]
		onlinePlayersMu.Unlock()

		if !ok {
			_, _ = fmt.Fprintln(player.Conn, PlayerNotFoundErr.Error())
			slog.Info(PlayerNotFoundErr.Error(), "player", pname, "command", subcommand, "subargs", subargs)
			return
		}

		inParty := player.getPlayerGroupInfo()
		if inParty == nil {
			_, _ = fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", pname, "command", subcommand, "subargs", subargs)
			return
		}

		err := inParty.GroupInvite(player, invitee)
		if err != nil {
			_, _ = fmt.Fprintln(player.Conn, err.Error())
			slog.Info(err.Error(), "player", pname, "command", subcommand, "subargs", subargs)
			return
		}

		_, _ = fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "GROUP", "subcommand", subcommand)

	case "JOIN":
		err := GroupJoin(subargs, player)
		if err != nil {
			slog.Info(err.Error(), "player", pname, "command", subcommand, "subargs", subargs)
			return
		}

		_, _ = fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "GROUP", "subcommand", subcommand)

	case "LEAVE":
		inParty := player.getPlayerGroupInfo()
		if inParty == nil {
			_, _ = fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", pname, "command", subcommand, "subargs", subargs)
			return
		}

		err := inParty.GroupLeave(player)

		if err != nil {
			slog.Info(err.Error(), "player", pname, "command", subcommand, "subargs", subargs)
			return
		}

		_, _ = fmt.Fprintln(player.Conn, "OK")
		slog.Info("SYS_MESSAGE", "player", pname, "message", "OK", "command", "GROUP", "subcommand", subcommand)

	default:
		_, _ = fmt.Fprintln(player.Conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "player", pname, "command", subcommand)
	}
}

// GroupCreate creates a new group led by the given player.
func GroupCreate(player *Player) (*Group, error) {
	pname := player.getPlayerName()
	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()
	if player.GroupInfo != nil {
		return nil, InGroupErr
	}

	newgroup := Group{
		GroupID:    "PT-" + pname,
		LeaderName: pname,
		Members: map[string]*Player{
			pname: player,
		},
	}

	player.GroupInfo = &newgroup

	partiesMu.Lock()
	defer partiesMu.Unlock()
	parties[newgroup.GroupID] = &newgroup

	return &newgroup, nil
}

// GroupInvite invites a player to the group, if the inviter is its leader.
func (g *Group) GroupInvite(inviter, player *Player) error {
	inviterName := inviter.getPlayerName()
	pname := player.getPlayerName()

	if inviterName == pname {
		return InvalidCommandErr
	}

	g.GroupMu.Lock()
	defer g.GroupMu.Unlock()

	if inviterName != g.LeaderName {
		return NotLeaderErr
	}

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	if player.GroupInfo != nil {
		return InGroupErr
	}

	_, _ = fmt.Fprintln(player.Conn, EvtPartyInvite(inviterName))
	slog.Info("SYS_MESSAGE", "player", pname, "message", EvtPartyInvite(inviterName))
	return nil
}

// GroupJoin adds a player to the group led by leaderName.
func GroupJoin(leaderName string, player *Player) error {
	partiesMu.Lock()
	g, ok := parties["PT-"+leaderName]
	partiesMu.Unlock()

	pname := player.getPlayerName()

	if !ok {
		_, _ = fmt.Fprintln(player.Conn, NotLeaderErr.Error())
		return NotLeaderErr
	}

	g.GroupMu.Lock()
	defer g.GroupMu.Unlock()

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	if player.GroupInfo != nil {
		_, _ = fmt.Fprintln(player.Conn, InGroupErr.Error())
		return InGroupErr
	}

	player.GroupInfo = g
	g.Members[pname] = player

	for k, m := range g.Members {
		if k != pname {
			_, _ = fmt.Fprintln(m.Conn, EvtPartyJoin(pname))
			slog.Info("SYS_MESSAGE", "player", m.getPlayerName(), "message", EvtPartyJoin(pname))
		}
	}

	return nil
}

// GroupLeave removes the player from the group, disbanding it if they were the leader.
func (g *Group) GroupLeave(player *Player) error {
	pname := player.getPlayerName()

	if g.LeaderName == pname {
		GroupDisband(g.LeaderName, g)
		return nil
	}

	g.GroupMu.Lock()
	defer g.GroupMu.Unlock()

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	if player.GroupInfo != g {
		_, _ = fmt.Fprintln(player.Conn, NotInGroupErr.Error())
		return NotInGroupErr
	}

	delete(g.Members, pname)
	player.GroupInfo = nil

	for _, m := range g.Members {
		_, _ = fmt.Fprintln(m.Conn, EvtPartyLeave(pname))
		slog.Info("SYS_MESSAGE", "player", m.getPlayerName(), "message", EvtPartyLeave(pname))
	}

	return nil
}

// GroupDisband removes a group and clears its members' group info.
func GroupDisband(leaderName string, g *Group) {
	g.GroupMu.Lock()
	defer g.GroupMu.Unlock()
	partiesMu.Lock()
	defer partiesMu.Unlock()

	delete(parties, g.GroupID)

	for _, m := range g.Members {
		m.PlayerMu.Lock()
		m.GroupInfo = nil
		channel := m.Conn
		memName := m.Username
		m.PlayerMu.Unlock()

		if leaderName != memName {
			_, _ = fmt.Fprintln(channel, EvtPartyLeave(g.LeaderName))
			_, _ = fmt.Fprintln(channel, EvtPartyLeave(memName))
			slog.Info("SYS_MESSAGE", "player", memName, "message", EvtPartyLeave(g.LeaderName))
			slog.Info("SYS_MESSAGE", "player", memName, "message", EvtPartyLeave(memName))
		}
	}
}
