package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/constants"
)

type trackedStatus struct {
	ID     string
	Status string
}

// getTrackedIssueStatus queries tracked issues and their status.
// Uses bd show --json which embeds resolved dependency details (including
// cross-rig status) in a single call. Falls back to bd dep list when
// the show response doesn't include embedded dependencies.
func getTrackedIssueStatus(beadsDir, convoyID string) []trackedStatus {
	if !convoyIDPattern.MatchString(convoyID) {
		return nil
	}

	// Fast path: bd show --json embeds dependencies with full details.
	// This avoids the expensive bd dep list cross-database resolution (3-6s).
	if tracked := getTrackedFromShow(beadsDir, convoyID); len(tracked) > 0 {
		return tracked
	}

	// Fallback: bd dep list for convoys without embedded deps.
	return getTrackedFromDepList(beadsDir, convoyID)
}

// getTrackedFromShow extracts tracked issue status from bd show --json
// embedded dependencies. Returns nil if no dependencies are embedded.
func getTrackedFromShow(beadsDir, convoyID string) []trackedStatus {
	ctx, cancel := context.WithTimeout(context.Background(), constants.BdSubprocessTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bd", "show", convoyID, "--json")
	cmd.Dir = beadsDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil
	}

	var results []struct {
		Dependencies []struct {
			ID             string `json:"id"`
			Status         string `json:"status"`
			DependencyType string `json:"dependency_type"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil || len(results) == 0 {
		return nil
	}

	var tracked []trackedStatus
	for _, dep := range results[0].Dependencies {
		if dep.DependencyType != "tracks" {
			continue
		}
		tracked = append(tracked, trackedStatus{
			ID:     beads.ExtractIssueID(dep.ID),
			Status: dep.Status,
		})
	}
	return tracked
}

// getTrackedFromDepList uses bd dep list for convoys without embedded deps.
func getTrackedFromDepList(beadsDir, convoyID string) []trackedStatus {
	ctx, cancel := context.WithTimeout(context.Background(), constants.BdSubprocessTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bd", "dep", "list", convoyID, "-t", "tracks", "--json")
	cmd.Dir = beadsDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil
	}

	var deps []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &deps); err != nil {
		return nil
	}

	var tracked []trackedStatus
	for _, dep := range deps {
		tracked = append(tracked, trackedStatus{
			ID:     beads.ExtractIssueID(dep.ID),
			Status: dep.Status,
		})
	}
	return tracked
}
