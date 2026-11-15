// Command license-sign makes a Nexul enterprise license key from its fields and the issuer's private key file. A key
// stops working at 00:00 UTC on its expiry date, the date the instance shows.
//
//	go run ./ee/cmd/license-sign -key <(op read op://Fleet/NEXUL_LICENSE_SIGNING_KEY/credential) \
//	  -licensee "Acme" -url https://nexul.example.com -expires 2027-11-11
//
// -days 45 instead of -expires sets the expiry that many days after today, UTC.
package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/otal-labs/nexul/ee/license"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "license-sign:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("license-sign", flag.ContinueOnError)
	keyFile := fs.String("key", "", "the issuer's ed25519 private key, a PKCS#8 PEM file")
	licensee := fs.String("licensee", "", "who the key is for")
	instanceURL := fs.String("url", "", "the instance's public URL, such as https://nexul.example.com")
	expires := fs.String("expires", "", "the date the key stops working, YYYY-MM-DD, at 00:00 UTC")
	days := fs.Int("days", 0, "instead of -expires: the key stops working at 00:00 UTC this many days after today")
	features := fs.String("features", joinFeatures(license.Features), "comma-separated features the key unlocks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keyFile == "" {
		return errors.New("-key is required")
	}
	priv, err := readPrivateKey(*keyFile)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	expiry, err := expiryOf(*expires, *days, now)
	if err != nil {
		return err
	}
	claims := license.Claims{
		Licensee:    strings.TrimSpace(*licensee),
		InstanceURL: strings.TrimSpace(*instanceURL),
		Features:    splitFeatures(*features),
		ExpiresAt:   expiry,
		IssuedAt:    now.Truncate(time.Second),
	}
	key, err := license.Sign(claims, priv)
	if err != nil {
		return err
	}
	if _, err := license.Parse(key, license.PublicKey()); err != nil {
		return fmt.Errorf("the private key is not the half of the issuer key this build checks against: %w", err)
	}
	fmt.Fprintf(os.Stderr, "%s, %s, %s: stops working %s (in %d days)\n", claims.Licensee, claims.InstanceURL,
		joinFeatures(claims.Features), expiry.Format("Mon 2 Jan 2006 15:04 MST"), int(expiry.Sub(now).Hours()/24))
	fmt.Println(key)
	return nil
}

// expiryOf is 00:00 UTC on the -expires date, or -days days after today's UTC date; a key that would already have
// expired is refused, since the instance refuses it too.
func expiryOf(expires string, days int, now time.Time) (time.Time, error) {
	if (expires == "") == (days == 0) {
		return time.Time{}, errors.New("set one of -expires or -days")
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	expiry := today.AddDate(0, 0, days)
	if expires != "" {
		parsed, err := time.Parse(time.DateOnly, expires)
		if err != nil {
			return time.Time{}, fmt.Errorf("-expires %q is not a YYYY-MM-DD date", expires)
		}
		expiry = parsed.UTC()
	}
	if !expiry.After(now) {
		return time.Time{}, fmt.Errorf("the key would stop working on %s, which has already begun in UTC", expiry.Format(time.DateOnly))
	}
	return expiry, nil
}

func readPrivateKey(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read -key: %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("-key holds no PEM block")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse -key: %w", err)
	}
	priv, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("-key is not an ed25519 key")
	}
	return priv, nil
}

func joinFeatures(fs []license.Feature) string {
	names := make([]string, len(fs))
	for i, f := range fs {
		names[i] = string(f)
	}
	return strings.Join(names, ",")
}

func splitFeatures(s string) []license.Feature {
	var out []license.Feature
	for name := range strings.SplitSeq(s, ",") {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, license.Feature(name))
		}
	}
	return out
}
