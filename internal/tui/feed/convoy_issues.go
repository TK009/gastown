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
func getTrackedIssueStatus(beadsDir, convoyID string) []trackedStatus {
	if !convoyIDPattern.MatchString(convoyID) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.BdSubprocessTimeout)
	defer cancel()

	// Query tracked issues using bd dep list (returns full issue details
	// including cross-rig status via prefix-based routing).
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
