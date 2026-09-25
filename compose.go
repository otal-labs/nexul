// Package nexul embeds the production compose file the nexul CLI installs.
package nexul

import _ "embed"

// Compose is docker-compose.yml, written into the install directory by nexul install and nexul update.
//
//go:embed docker-compose.yml
var Compose []byte
