package main

import (
	"fmt"
	"log/slog"
	"sync"
)

var (
	parties   = make(map[string]*Group)
	partiesMu sync.Mutex
)

type Group struct {
	GroupID    string
	LeaderName string
	Members    map[string]*Player
	GroupMu    sync.Mutex
}

func groupFuncDispatcher(subcommand, subargs string, player *Player) {
	switch subcommand {
	case "CREATE":
		group, err := GroupCreate(player)
		if err != nil {
			fmt.Fprintln(player.Conn, err.Error())
			slog.Info(err.Error(), "player", player.Username, "command", subcommand, "subargs", subargs)
			return
		}
		fmt.Fprintln(player.Conn, "OK group="+group.GroupID)
	case "INVITE":
		onlinePlayersMu.Lock()
		invitee, ok := onlinePlayers[subargs]
		onlinePlayersMu.Unlock()
		if !ok {
			fmt.Fprintln(player.Conn, PlayerNotFoundErr.Error())
			slog.Info(PlayerNotFoundErr.Error(), "player", player.Username, "command", subcommand, "subargs", subargs)
			return
		}
		player.PlayerMu.Lock()
		inParty := player.GroupInfo
		player.PlayerMu.Unlock()
		if inParty == nil {
			fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", player.Username, "command", subcommand, "subargs", subargs)
			return
		}
		err := inParty.GroupInvite(player, invitee)

		if err != nil {
			slog.Info(err.Error(), "player", player.Username, "command", subcommand, "subargs", subargs)
		}
	case "JOIN":
		err := GroupJoin(subargs, player)

		if err != nil {
			slog.Info(err.Error(), "player", player.Username, "command", subcommand, "subargs", subargs)
		}
	case "LEAVE":
		player.PlayerMu.Lock()
		inParty := player.GroupInfo
		player.PlayerMu.Unlock()
		if inParty == nil {
			fmt.Fprintln(player.Conn, NotInGroupErr.Error())
			slog.Info(NotInGroupErr.Error(), "player", player.Username, "command", subcommand, "subargs", subargs)
			return
		}

		err := inParty.GroupLeave(player)

		if err != nil {
			slog.Info(err.Error(), "player", player.Username, "command", subcommand, "subargs", subargs)
		}
	default:
		fmt.Fprintln(player.Conn, InvalidCommandErr.Error())
		slog.Info(InvalidCommandErr.Error(), "player", player.Username, "command", subcommand)
	}
}

func GroupCreate(player *Player) (*Group, error) {
	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()
	if player.GroupInfo != nil {
		return nil, InGroupErr
	}

	newgroup := Group{
		GroupID:    "PT-" + player.Username,
		LeaderName: player.Username,
		Members: map[string]*Player{
			player.Username: player,
		},
	}

	player.GroupInfo = &newgroup

	partiesMu.Lock()
	defer partiesMu.Unlock()
	parties[newgroup.GroupID] = &newgroup

	return &newgroup, nil
}

func (g *Group) GroupInvite(inviter, player *Player) error {
	if inviter.Username == player.Username {
		fmt.Fprintln(inviter.Conn, InvalidCommandErr.Error())
		return InvalidCommandErr
	}

	g.GroupMu.Lock()
	defer g.GroupMu.Unlock()

	if inviter.Username != g.LeaderName {
		fmt.Fprintln(inviter.Conn, NotLeaderErr.Error())
		return NotLeaderErr
	}

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	if player.GroupInfo != nil {
		fmt.Fprintln(inviter.Conn, InGroupErr.Error())
		return InGroupErr
	}

	fmt.Fprintln(player.Conn, EvtPartyInvite(inviter.Username))
	return nil
}

func GroupJoin(leaderName string, player *Player) error {
	partiesMu.Lock()
	g, ok := parties["PT-"+leaderName]
	partiesMu.Unlock()

	if !ok {
		fmt.Fprintln(player.Conn, NotLeaderErr.Error())
		return NotLeaderErr
	}

	g.GroupMu.Lock()
	defer g.GroupMu.Unlock()

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	if player.GroupInfo != nil {
		fmt.Fprintln(player.Conn, InGroupErr.Error())
		return InGroupErr
	}

	player.GroupInfo = g
	g.Members[player.Username] = player

	for k, m := range g.Members {
		if k == player.Username {
			continue
		}
		fmt.Fprintln(m.Conn, EvtPartyJoin(player.Username))
	}

	return nil
}

func (g *Group) GroupLeave(player *Player) error {
	if g.LeaderName == player.Username {
		GroupDisband(g.LeaderName, g)
		return nil
	}

	g.GroupMu.Lock()
	defer g.GroupMu.Unlock()

	player.PlayerMu.Lock()
	defer player.PlayerMu.Unlock()

	if player.GroupInfo != g {
		fmt.Fprintln(player.Conn, NotInGroupErr.Error())
		return NotInGroupErr
	}

	delete(g.Members, player.Username)
	player.GroupInfo = nil

	for _, m := range g.Members {
		fmt.Fprintln(m.Conn, EvtPartyLeave(player.Username))
	}

	return nil
}

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
		pname := m.Username
		m.PlayerMu.Unlock()

		if leaderName != pname {
			fmt.Fprintln(channel, EvtPartyLeave(g.LeaderName))
			fmt.Fprintln(channel, EvtPartyLeave(pname))
		}
	}
}
