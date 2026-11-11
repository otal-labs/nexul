// Command license-sign makes a Nexul enterprise license key from its fields and the issuer's private key file.
//
//	go run ./ee/cmd/license-sign -key <(op read op://Fleet/NEXUL_LICENSE_SIGNING_KEY/credential) \
//	  -licensee "Acme" -url https://nexul.example.com -expires 2027-11-11
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
	expiry, err := time.Parse(time.DateOnly, *expires)
	if err != nil {
		return fmt.Errorf("-expires %q is not a YYYY-MM-DD date", *expires)
	}
	claims := license.Claims{
		Licensee:    strings.TrimSpace(*licensee),
		InstanceURL: strings.TrimSpace(*instanceURL),
		Features:    splitFeatures(*features),
		ExpiresAt:   expiry.UTC(),
		IssuedAt:    time.Now().UTC().Truncate(time.Second),
	}
	key, err := license.Sign(claims, priv)
	if err != nil {
		return err
	}
	if _, err := license.Parse(key, license.PublicKey()); err != nil {
		return fmt.Errorf("the private key is not the half of the issuer key this build checks against: %w", err)
	}
	fmt.Println(key)
	return nil
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
