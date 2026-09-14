package main

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// softbannedPlayers holds soft-banned hosts and when their ban expires.
// spammingIPConns tracks recent connection timestamps per host, for flood detection.
var (
	softbannedPlayers   = make(map[string]time.Time)
	softbannedPlayersMu sync.Mutex
	recheckSoftbanCh    = make(chan struct{})

	spammingIPConns = make(map[string][]time.Time)
	spammingIPMu    sync.Mutex
)

// isOnTimeout reports whether the player is currently timed out, and for how much longer.
func (p *Player) isOnTimeout(timestamp time.Time) (bool, time.Duration) {
	p.PlayerMu.Lock()
	currTimeout := p.TimeoutEnd
	p.PlayerMu.Unlock()

	if timestamp.Before(currTimeout) {
		return true, currTimeout.Sub(timestamp)
	}

	return false, 0
}

// handleTimeoutBucket applies the player's per-input token bucket, timing them out if it's empty.
func (p *Player) handleTimeoutBucket(timestamp time.Time) bool {
	const refill = 0.75
	const maxCapacity = 8

	p.PlayerMu.Lock()
	defer p.PlayerMu.Unlock()
	timeElapsed := timestamp.Sub(p.BucketTS).Seconds()
	tokensToAdd := timeElapsed * refill
	p.TokenCount = min(p.TokenCount+tokensToAdd, maxCapacity)
	p.BucketTS = timestamp

	if p.TokenCount >= 1 {
		p.TokenCount -= 1
		return true
	}

	p.TimeoutEnd = timestamp.Add(5 * time.Minute)
	p.TimeoutActions = 0

	return false
}

// isIPSoftbanned reports whether the given IP is currently soft-banned.
func isIPSoftbanned(ip string, timestamp time.Time) (bool, time.Time) {
	softbannedPlayersMu.Lock()
	untilWhen, banned := softbannedPlayers[ip]
	softbannedPlayersMu.Unlock()

	if banned {
		stillBanned := timestamp.Before(untilWhen)

		if stillBanned {
			return stillBanned, untilWhen
		} else {
			select {
			case recheckSoftbanCh <- struct{}{}:
			default:
			}

		}
	}

	return false, time.Time{}
}

// handleSoftban warns the player or, after enough strikes, soft-bans their IP.
func (p *Player) handleSoftban(timeRemaining time.Duration) {
	p.PlayerMu.Lock()
	p.TimeoutActions += 1
	actions := p.TimeoutActions
	pConn := p.Conn
	p.PlayerMu.Unlock()

	if actions < 10 {
		_, _ = fmt.Fprintf(p.Conn, "%v. Time remaining: %v\n", InputSpamErr.Error(), timeRemaining.Round(time.Second))
		slog.Warn("SYS_MESSAGE", "remote", p.Conn.RemoteAddr().String(), "player", p.Username, "message", InputSpamErr.Error(), "time_remaining", timeRemaining.Round(time.Second))
	} else {
		_, _ = fmt.Fprintln(p.Conn, SoftbannedErr.Error())
		_ = p.Conn.Close()

		softbannedPlayersMu.Lock()
		host, _, _ := net.SplitHostPort(pConn.RemoteAddr().String())
		softbannedPlayers[host] = time.Now().Add(20 * time.Minute)
		softbannedPlayersMu.Unlock()

		slog.Warn(SoftbannedErr.Error(), "remote", p.Conn.RemoteAddr().String(), "player", p.Username, "message", SoftbannedErr.Error(), "time_remaining", timeRemaining.Round(time.Second))
	}
}

// cleanSoftbanList removes expired entries from softbannedPlayers.
func cleanSoftbanList() {
	softbannedPlayersMu.Lock()
	for ip, until := range softbannedPlayers {
		if time.Now().After(until) {
			delete(softbannedPlayers, ip)
		}
	}
	softbannedPlayersMu.Unlock()
}

// trackConnCount records a connection from host and soft-bans it if it's flooding.
func trackConnCount(conn net.Conn, host string) {
	now := time.Now()

	spammingIPMu.Lock()
	spammingIPConns[host] = append(spammingIPConns[host], now)
	timeList := spammingIPConns[host]

	var newTList []time.Time

	for _, t := range timeList {
		if now.Sub(t).Seconds() <= 5 {
			newTList = append(newTList, t)
		}
	}

	spammingIPConns[host] = newTList
	spammingIPMu.Unlock()

	if len(newTList) >= 5 {
		_, _ = fmt.Fprintln(conn, SoftbannedErr.Error())
		_ = conn.Close()

		softbannedPlayersMu.Lock()
		timeRemaining := time.Duration(generateRandInt(20, 30)) * time.Minute
		softbannedPlayers[host] = time.Now().Add(timeRemaining.Round(time.Second))
		softbannedPlayersMu.Unlock()

		slog.Warn(SoftbannedErr.Error(), "remote", host, "message", SoftbannedErr.Error(), "time_remaining", timeRemaining.Round(time.Second))
	}
}
