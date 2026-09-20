package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// composeDoc decodes only the "services" key, as a raw node so declaration order (needed for Reachable) survives —
// Go map iteration order is random, and compose's short/long forms mix scalars and mappings under one key.
type composeDoc struct {
	Services yaml.Node `yaml:"services"`
}

// composeServiceRaw mirrors one compose service, keeping the variant-shaped fields as raw nodes.
type composeServiceRaw struct {
	Image       string    `yaml:"image"`
	Build       yaml.Node `yaml:"build"`
	Ports       yaml.Node `yaml:"ports"`
	Expose      yaml.Node `yaml:"expose"`
	Environment yaml.Node `yaml:"environment"`
	EnvFile     yaml.Node `yaml:"env_file"`
}

// errUnparsable marks a compose file whose YAML does not parse: it is simply not a candidate, and the scan
// carries on with the rest of the tree instead of failing on one broken example file.
var errUnparsable = errors.New("unparsable")

// scanComposeCandidate fetches and parses one compose file into a Candidate.
func scanComposeCandidate(ctx context.Context, s Scanner, owner, name, ref, repoName, filePath string) (Candidate, error) {
	b, err := s.GetFile(ctx, owner, name, ref, filePath)
	if err != nil {
		return Candidate{}, fmt.Errorf("get compose file %s: %w", filePath, err)
	}
	services, err := parseCompose(b)
	if err != nil {
		return Candidate{}, fmt.Errorf("%w: compose file %s: %w", errUnparsable, filePath, err)
	}
	return Candidate{
		Kind:      KindCompose,
		Path:      filePath,
		Name:      candidateName(repoName, filePath),
		Services:  services,
		Reachable: firstReachable(services),
	}, nil
}

// parseCompose extracts every service's name, image/build, ports, expose and env keys, in declaration order.
func parseCompose(b []byte) ([]DeclaredService, error) {
	var doc composeDoc
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Services.Kind != yaml.MappingNode {
		return nil, nil
	}
	out := make([]DeclaredService, 0, len(doc.Services.Content)/2)
	for i := 0; i+1 < len(doc.Services.Content); i += 2 {
		svcName := doc.Services.Content[i].Value
		// A mis-indented file leaves a service as a bare key (null) with its settings hoisted a level up; that
		// entry is not a service, and the rest of the file may still be.
		if doc.Services.Content[i+1].Kind != yaml.MappingNode {
			continue
		}
		var raw composeServiceRaw
		if err := doc.Services.Content[i+1].Decode(&raw); err != nil {
			return nil, fmt.Errorf("service %s: %w", svcName, err)
		}
		out = append(out, DeclaredService{
			Name:    svcName,
			Image:   raw.Image,
			Build:   parseBuild(raw.Build),
			Ports:   parsePorts(raw.Ports),
			Expose:  parsePorts(raw.Expose),
			EnvKeys: parseServiceEnvKeys(raw.Environment, raw.EnvFile),
		})
	}
	return out, nil
}

// parseBuild reads compose's short form (`build: ./dir`) or long form (`build: {context, dockerfile}`).
func parseBuild(n yaml.Node) *Build {
	switch n.Kind {
	case yaml.ScalarNode:
		return &Build{Context: n.Value}
	case yaml.MappingNode:
		b := &Build{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i].Value, n.Content[i+1].Value
			switch key {
			case "context":
				b.Context = val
			case "dockerfile":
				b.Dockerfile = val
			}
		}
		return b
	default:
		return nil
	}
}

// parsePorts reads compose's ports/expose list — short syntax strings ("8080:80") and long syntax mappings
// ({target: 80}) — and returns the container-side port from each entry.
func parsePorts(n yaml.Node) []int {
	if n.Kind != yaml.SequenceNode {
		return nil
	}
	var ports []int
	for _, item := range n.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			if p, ok := parseShortPort(item.Value); ok {
				ports = append(ports, p)
			}
		case yaml.MappingNode:
			for i := 0; i+1 < len(item.Content); i += 2 {
				if item.Content[i].Value != "target" {
					continue
				}
				if p, ok := parsePortToken(item.Content[i+1].Value); ok {
					ports = append(ports, p)
				}
			}
		}
	}
	return ports
}

// parseShortPort takes the last ":"-separated segment of a short-syntax port entry — the container port,
// whether the entry is bare ("3000"), "host:container", or "ip:host:container".
func parseShortPort(s string) (int, bool) {
	parts := strings.Split(s, ":")
	return parsePortToken(parts[len(parts)-1])
}

// parseEnvironment reads compose's map form ({KEY: value}) or list form (["KEY=value", "KEY"]), key names only.
func parseEnvironment(n yaml.Node) []string {
	switch n.Kind {
	case yaml.MappingNode:
		keys := make([]string, 0, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			keys = append(keys, n.Content[i].Value)
		}
		return keys
	case yaml.SequenceNode:
		keys := make([]string, 0, len(n.Content))
		for _, item := range n.Content {
			if item.Kind != yaml.ScalarNode {
				continue
			}
			key := item.Value
			if i := strings.Index(key, "="); i >= 0 {
				key = key[:i]
			}
			keys = append(keys, key)
		}
		return keys
	default:
		return nil
	}
}

// parseEnvFileNames reads compose's string or list form of env_file, keeping only the referenced file names —
// their contents aren't fetched, so the wizard surfaces the file name itself as a placeholder env entry.
func parseEnvFileNames(n yaml.Node) []string {
	switch n.Kind {
	case yaml.ScalarNode:
		return []string{n.Value}
	case yaml.SequenceNode:
		names := make([]string, 0, len(n.Content))
		for _, item := range n.Content {
			if item.Kind == yaml.ScalarNode {
				names = append(names, item.Value)
			}
		}
		return names
	default:
		return nil
	}
}

func parseServiceEnvKeys(environment, envFile yaml.Node) []string {
	keys := parseEnvironment(environment)
	return append(keys, parseEnvFileNames(envFile)...)
}
