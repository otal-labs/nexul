package repository

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
)

// scanDockerfileCandidate fetches and parses one standalone Dockerfile into a single-service Candidate.
func scanDockerfileCandidate(ctx context.Context, s Scanner, owner, name, ref, repoName, filePath string) (Candidate, error) {
	b, err := s.GetFile(ctx, owner, name, ref, filePath)
	if err != nil {
		return Candidate{}, fmt.Errorf("get dockerfile %s: %w", filePath, err)
	}
	svcName := candidateName(repoName, filePath)
	svc := DeclaredService{
		Name:  svcName,
		Build: &Build{Context: path.Dir(filePath), Dockerfile: path.Base(filePath)},
		// EXPOSE only documents a port; nothing publishes it without a compose file, so it lands in Expose, not Ports.
		Expose: parseDockerfileExpose(b),
	}
	return Candidate{
		Kind:      KindDockerfile,
		Path:      filePath,
		Name:      svcName,
		Services:  []DeclaredService{svc},
		Reachable: firstReachable([]DeclaredService{svc}),
	}, nil
}

// parseDockerfileExpose extracts every port named on an EXPOSE instruction, in file order.
func parseDockerfileExpose(b []byte) []int {
	var ports []int
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		fields := strings.Fields(strings.TrimSpace(scanner.Text()))
		if len(fields) < 2 || !strings.EqualFold(fields[0], "EXPOSE") {
			continue
		}
		for _, tok := range fields[1:] {
			if p, ok := parsePortToken(tok); ok {
				ports = append(ports, p)
			}
		}
	}
	return ports
}
