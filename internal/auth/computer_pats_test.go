package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

func TestTokens_HidesAMintedToken(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)

	raw, _, err := s.MintPAT(context.Background(), u, "ci agent")
	require.NoError(t, err)
	assert.Equal(t, "wrote "+redact.Placeholder, redact.Tokens("wrote "+raw))
}

func TestMintComputerPAT_ErrorsFirst(t *testing.T) {
	s, users, pats := newPATHarness()
	u := seedPATUser(t, users)

	_, _, err := s.MintComputerPAT(context.Background(), u, "  ", "Laptop")
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	_, _, err = s.MintComputerPAT(context.Background(), "ghost", "c1", "Laptop")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	pats.getErr = errors.New("disk gone")
	_, _, err = s.MintComputerPAT(context.Background(), u, "c1", "Laptop")
	require.ErrorContains(t, err, "disk gone")
}

func TestMintComputerPAT_NamesItAfterTheComputerAndReplacesTheActiveOne(t *testing.T) {
	s, users, pats := newPATHarness()
	u := seedPATUser(t, users)
	ctx := context.Background()

	firstRaw, first, err := s.MintComputerPAT(ctx, u, "c1", " Laptop ")
	require.NoError(t, err)
	assert.Equal(t, "Nexul MCP on Laptop", first.Name)
	assert.Equal(t, "c1", first.ComputerID)

	_, second, err := s.MintComputerPAT(ctx, u, "c1", "Laptop")
	require.NoError(t, err)

	active, err := s.ComputerPAT(ctx, u, "c1")
	require.NoError(t, err)
	assert.Equal(t, second.ID, active.ID)
	_, err = s.AuthenticatePAT(ctx, firstRaw)
	require.ErrorIs(t, err, apperrs.ErrUnauthorized, "the replaced token stops working")
	assert.Equal(t, []string{TopicTokenMinted, TopicTokenRevoked, TopicTokenMinted}, pats.topics())
}

func TestRevokeComputerPAT(t *testing.T) {
	s, users, pats := newPATHarness()
	u := seedPATUser(t, users)
	ctx := context.Background()

	require.NoError(t, s.RevokeComputerPAT(ctx, u, "c1"), "a computer without a token is already revoked")
	assert.Empty(t, pats.topics())

	raw, _, err := s.MintComputerPAT(ctx, u, "c1", "Laptop")
	require.NoError(t, err)
	require.NoError(t, s.RevokeComputerPAT(ctx, u, "c1"))
	_, err = s.AuthenticatePAT(ctx, raw)
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, err = s.ComputerPAT(ctx, u, "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, []string{TopicTokenMinted, TopicTokenRevoked}, pats.topics())

	pats.getErr = errors.New("disk gone")
	require.ErrorContains(t, s.RevokeComputerPAT(ctx, u, "c1"), "disk gone")
}

func TestRevokePAT_UnknownTokenIsNotFoundAndWritesNoEvent(t *testing.T) {
	s, users, pats := newPATHarness()
	u := seedPATUser(t, users)

	require.ErrorIs(t, s.RevokePAT(context.Background(), u, "pat-missing"), apperrs.ErrNotFound)
	assert.Empty(t, pats.topics())
}
