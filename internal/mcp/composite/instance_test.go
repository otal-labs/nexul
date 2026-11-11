package composite

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/license"
	"github.com/otal-labs/nexul/ee/sso"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type fakeLicenseKeeper struct{ key string }

func (f *fakeLicenseKeeper) status() license.Status {
	if f.key == "" {
		return license.Status{State: license.StateNone}
	}
	return license.Status{State: license.StateActive, Licensee: "Acme"}
}
func (f *fakeLicenseKeeper) Status(context.Context) (license.Status, error) { return f.status(), nil }
func (f *fakeLicenseKeeper) Set(_ context.Context, key string) (license.Status, error) {
	if key == "bad" {
		return license.Status{}, apperrs.ErrInvalid
	}
	f.key = key
	return f.status(), nil
}
func (f *fakeLicenseKeeper) Remove(context.Context) (license.Status, error) {
	f.key = ""
	return f.status(), nil
}

type fakeSSOSettings struct {
	view    sso.View
	patches []sso.Patch
	err     error
}

func (f *fakeSSOSettings) View(context.Context) (sso.View, error) { return f.view, nil }
func (f *fakeSSOSettings) Update(_ context.Context, p sso.Patch) (sso.View, error) {
	if f.err != nil {
		return sso.View{}, f.err
	}
	f.patches = append(f.patches, p)
	f.view.Configured = true
	return f.view, nil
}
func (f *fakeSSOSettings) Remove(context.Context) (sso.View, error) {
	f.view.Configured = false
	return f.view, nil
}

func instanceCall(t *testing.T, l LicenseKeeper, s SSOSettings, args string) (instanceSettings, error) {
	t.Helper()
	out, err := call(t, t.Context(), InstanceTools(l, s), "instance_update", args)
	if err != nil {
		return instanceSettings{}, err
	}
	return out.(instanceSettings), nil
}

func TestInstanceUpdate_ReadsBothSettingsBackWhateverItChanged(t *testing.T) {
	t.Parallel()
	l, s := &fakeLicenseKeeper{}, &fakeSSOSettings{}

	got, err := instanceCall(t, l, s, `{}`)
	require.NoError(t, err)
	assert.Equal(t, license.StateNone, got.License.State, "an omitted field changes nothing")

	got, err = instanceCall(t, l, s, `{"license_key":"key-1"}`)
	require.NoError(t, err)
	assert.Equal(t, license.StateActive, got.License.State)
	assert.False(t, got.SSO.Configured, "the sso settings are read back, not skipped")

	got, err = instanceCall(t, l, s, `{"license_key":""}`)
	require.NoError(t, err)
	assert.Equal(t, license.StateNone, got.License.State)
	assert.Empty(t, l.key)
}

func TestInstanceUpdate_SSOFieldsPassThroughAsAPatch(t *testing.T) {
	t.Parallel()
	l, s := &fakeLicenseKeeper{key: "k"}, &fakeSSOSettings{}

	got, err := instanceCall(t, l, s, `{"sso":{"display_name":"Acme SSO","only":false}}`)
	require.NoError(t, err)
	assert.True(t, got.SSO.Configured)
	require.Len(t, s.patches, 1)
	assert.Equal(t, "Acme SSO", *s.patches[0].DisplayName)
	assert.False(t, *s.patches[0].Only, "false is a value, not an omission")
	assert.Nil(t, s.patches[0].IssuerURL, "an omitted field is left alone")
	assert.Nil(t, s.patches[0].ClientSecret)

	got, err = instanceCall(t, l, s, `{"sso":{"remove":true,"display_name":"ignored"}}`)
	require.NoError(t, err)
	assert.False(t, got.SSO.Configured)
	assert.Len(t, s.patches, 1, "remove ignores the other fields")
}

func TestInstanceUpdate_ASettingThatFailsAfterAnotherApplied_SaysWhatWasChanged(t *testing.T) {
	t.Parallel()
	l, s := &fakeLicenseKeeper{}, &fakeSSOSettings{err: apperrs.ErrInvalid}

	_, err := instanceCall(t, l, s, `{"license_key":"key-1","sso":{"display_name":"x"}}`)

	var partial *mcptool.PartialError
	require.True(t, errors.As(err, &partial), "%v", err)
	assert.Equal(t, []string{"license_key"}, partial.Applied)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Equal(t, "key-1", l.key, "the key was saved before the provider was refused")

	_, err = instanceCall(t, &fakeLicenseKeeper{}, &fakeSSOSettings{}, `{"license_key":"bad","sso":{"display_name":"x"}}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "nothing was changed", "a first step that fails changes nothing")
}

func TestInstanceUpdate_AnUnknownSSOFieldIsRefused(t *testing.T) {
	t.Parallel()
	_, err := instanceCall(t, &fakeLicenseKeeper{}, &fakeSSOSettings{}, `{"sso":{"client_secrt":"typo"}}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a mistyped field is never taken for an omitted one")
}
