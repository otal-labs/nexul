package issuesync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// keyedLocks holds one lock per key, kept only while someone holds or waits on it.
type keyedLocks struct {
	mu   sync.Mutex
	held map[string]*keyedLock
}

// keyedLock is a one-slot channel, not a mutex, so a goroutine waiting on it counts as blocked under testing/synctest.
type keyedLock struct {
	slot  chan struct{}
	users int
}

// lock blocks until key is free and returns the call that frees it.
func (k *keyedLocks) lock(key string) func() {
	k.mu.Lock()
	if k.held == nil {
		k.held = map[string]*keyedLock{}
	}
	l := k.held[key]
	if l == nil {
		l = &keyedLock{slot: make(chan struct{}, 1)}
		k.held[key] = l
	}
	l.users++
	k.mu.Unlock()
	l.slot <- struct{}{}
	return func() {
		<-l.slot
		k.mu.Lock()
		defer k.mu.Unlock()
		l.users--
		if l.users == 0 {
			delete(k.held, key)
		}
	}
}

// issueKey names an issue's lock; GitHub matches owner and repository names in any case.
func issueKey(owner, repo string, number int) string {
	return fmt.Sprintf("issue %s/%s#%d", strings.ToLower(owner), strings.ToLower(repo), number)
}

// ticketKey names the lock of a ticket being made into an issue, which has no number to lock yet.
func ticketKey(ticketID string) string { return "ticket " + ticketID }

// lockIssue locks i, and the ticket its marker names, so applying it never races that ticket's issue being made.
func (s *Service) lockIssue(owner, repo string, i Issue) func() {
	unlock := s.locks.lock(issueKey(owner, repo, i.Number))
	ticketID := markedTicket(i.Body)
	if ticketID == "" {
		return unlock
	}
	unlockTicket := s.locks.lock(ticketKey(ticketID))
	return func() {
		unlockTicket()
		unlock()
	}
}

// lockTicket locks the issue ticketID is linked to, or the ticket while it has none, and returns the link as read
// under that lock; a link that moved between the read and the lock is read and locked again.
func (s *Service) lockTicket(ctx context.Context, ticketID string) (func(), Link, bool, error) {
	link, linked, err := s.linkOf(ctx, ticketID)
	for err == nil {
		key := lockKeyOf(ticketID, link, linked)
		unlock := s.locks.lock(key)
		link, linked, err = s.linkOf(ctx, ticketID)
		if err == nil && lockKeyOf(ticketID, link, linked) == key {
			return unlock, link, linked, nil
		}
		unlock()
	}
	return nil, Link{}, false, err
}

func (s *Service) linkOf(ctx context.Context, ticketID string) (Link, bool, error) {
	link, err := s.d.Repo.GetLinkByTicket(ctx, ticketID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return Link{}, false, nil
	}
	return link, err == nil, err
}

func lockKeyOf(ticketID string, link Link, linked bool) string {
	if !linked {
		return ticketKey(ticketID)
	}
	return issueKey(link.Owner, link.Repo, link.IssueNumber)
}
