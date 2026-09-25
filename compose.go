// Package nexul embeds the compose files the nexul CLI installs.
package nexul

import _ "embed"

// Compose is docker-compose.yml, written into the install directory by nexul install and nexul upgrade.
//
//go:embed docker-compose.yml
var Compose []byte

// DesktopCompose is docker-compose.desktop.yml, layered over Compose on macOS and Windows.
//
//go:embed docker-compose.desktop.yml
var DesktopCompose []byte
