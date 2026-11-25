package auditlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/webhooksig"
)

const deliveryTimeout = 15 * time.Second

var errNoDestination = fmt.Errorf("%w: the stream has no stream_url to send to", apperrs.ErrInvalid)

// refuseRedirect keeps a 3xx as the answer, so a destination that moved fails visibly instead of being followed.
func refuseRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// Test posts one signed sample row to the destination in p, or the saved one, and returns what the receiver
// answered; nothing is queued. It needs what Update does.
func (s *Service) Test(ctx context.Context, p Patch) (TestResult, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return TestResult{}, err
	}
	if !s.unlocked(ctx) {
		return TestResult{}, errNeedsLicense
	}
	st, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return TestResult{}, err
	}
	if u := trim(p.StreamURL); u != "" {
		st.StreamURL = u
	}
	if secret := trim(p.StreamSecret); secret != "" {
		if err := s.checkSecret(ctx, secret); err != nil {
			return TestResult{}, err
		}
		st.StreamSecret = secret
	}
	if st.StreamURL == "" {
		return TestResult{}, errNoDestination
	}
	if err := checkStreamURL(st.StreamURL); err != nil {
		return TestResult{}, err
	}
	sample := Pending{ID: "test-" + ids.New(), ActorType: "user", ActorID: actorID(ctx), Action: "test", CreatedAt: s.cfg.Now().UTC()}
	status, err := s.send(ctx, st, sample)
	if err != nil {
		return TestResult{Status: status, Error: err.Error()}, nil
	}
	return TestResult{Status: status}, nil
}

// RetryFailed returns every dead delivery to pending, for the relay to send again. It needs what Update does.
func (s *Service) RetryFailed(ctx context.Context) (View, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return View{}, err
	}
	if !s.unlocked(ctx) {
		return View{}, errNeedsLicense
	}
	if _, err := s.cfg.Queue.RetryDead(ctx, s.cfg.Now()); err != nil {
		return View{}, err
	}
	return s.View(ctx)
}

// RotateSecret replaces the signing secret with one the server makes, returned once; deliveries from then on carry
// its signature. It needs what Update does and a saved destination.
func (s *Service) RotateSecret(ctx context.Context) (Saved, error) {
	if err := s.require(ctx, permissions.InstanceWrite); err != nil {
		return Saved{}, err
	}
	if !s.unlocked(ctx) {
		return Saved{}, errNeedsLicense
	}
	cur, err := s.cfg.Repo.Get(ctx)
	if err != nil {
		return Saved{}, err
	}
	if cur.StreamURL == "" {
		return Saved{}, errNoDestination
	}
	made, err := newSecret()
	if err != nil {
		return Saved{}, err
	}
	next := cur
	next.StreamSecret, next.UpdatedBy, next.UpdatedAt = made, actorID(ctx), s.cfg.Now()
	evt := s.changed(SettingsChangedEvent{
		RetentionDays: next.Days(), StreamURL: next.StreamURL, StreamEnabled: next.StreamEnabled, ChangedBy: next.UpdatedBy,
	}, next.UpdatedAt)
	if err := s.cfg.Repo.Set(ctx, next, evt); err != nil {
		return Saved{}, err
	}
	v, err := s.view(ctx, next)
	return Saved{View: v, NewSecret: made}, err
}

// send posts one row to st's destination, signed with its secret, and returns the receiver's status; any answer
// but a 2xx is an error naming it.
func (s *Service) send(ctx context.Context, st Settings, row Pending) (int, error) {
	secret, err := s.secretValue(ctx, st.StreamSecret)
	if err != nil {
		return 0, err
	}
	instanceURL, err := s.cfg.InstanceURL(ctx)
	if err != nil {
		return 0, fmt.Errorf("read instance url: %w", err)
	}
	body, err := json.Marshal(delivery{
		DeliveryID: ids.New(), ID: row.ID, CreatedAt: row.CreatedAt.UTC(), ActorType: row.ActorType, ActorID: row.ActorID,
		TokenID: row.TokenID, Action: row.Action, InstanceURL: instanceURL,
	})
	if err != nil {
		return 0, fmt.Errorf("encode audit delivery: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, st.StreamURL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build audit delivery: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(webhooksig.Header, webhooksig.Sign(secret, body))
	resp, err := s.cfg.Client.Do(req)
	if err != nil {
		return 0, reachError(err)
	}
	defer func() { _ = resp.Body.Close() }() // the body was read or is not needed; a close error changes nothing
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.StatusCode, fmt.Errorf("the destination answered %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

// reachError says why no answer came, without the URL, which a client error repeats.
func reachError(err error) error {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		if uerr.Timeout() {
			return fmt.Errorf("the destination did not answer within %s", deliveryTimeout)
		}
		err = uerr.Err
	}
	return fmt.Errorf("the destination could not be reached: %w", err)
}

func (s *Service) secretValue(ctx context.Context, stored string) (string, error) {
	if s.cfg.Secrets == nil {
		return stored, nil
	}
	v, err := s.cfg.Secrets.Value(ctx, stored)
	if err != nil {
		return "", fmt.Errorf("resolve the audit stream secret: %w", err)
	}
	return v, nil
}
