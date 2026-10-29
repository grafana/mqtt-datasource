package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// TeamChecker decides whether a Grafana user belongs to one of the teams
// allowed to publish.
type TeamChecker interface {
	IsMember(ctx context.Context, login string) (bool, error)
	// Verify bypasses the cache and checks that all configured teams can be read.
	Verify(ctx context.Context) error
}

const teamAPITimeout = 5 * time.Second

// GrafanaTeamChecker reads team membership through the Grafana HTTP API using
// a service account token. Successful lookups are cached; failures are not, so
// callers fail closed on API errors.
type GrafanaTeamChecker struct {
	baseURL    string
	token      string
	teams      []string
	ttl        time.Duration
	httpClient *http.Client
	now        func() time.Time

	mu        sync.Mutex
	members   map[string]struct{}
	expiresAt time.Time
}

func NewGrafanaTeamChecker(baseURL, token string, teams []string, ttl time.Duration) *GrafanaTeamChecker {
	return &GrafanaTeamChecker{
		baseURL:    baseURL,
		token:      token,
		teams:      teams,
		ttl:        ttl,
		httpClient: &http.Client{Timeout: teamAPITimeout},
		now:        time.Now,
	}
}

func (c *GrafanaTeamChecker) IsMember(ctx context.Context, login string) (bool, error) {
	if login == "" {
		return false, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.members == nil || !c.now().Before(c.expiresAt) {
		if err := c.refreshLocked(ctx); err != nil {
			return false, err
		}
	}
	_, ok := c.members[login]
	return ok, nil
}

func (c *GrafanaTeamChecker) Verify(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.refreshLocked(ctx)
}

func (c *GrafanaTeamChecker) refreshLocked(ctx context.Context) error {
	members := make(map[string]struct{})
	for _, team := range c.teams {
		teamID, err := c.lookupTeamID(ctx, team)
		if err != nil {
			c.members = nil
			return err
		}
		logins, err := c.teamMembers(ctx, teamID)
		if err != nil {
			c.members = nil
			return err
		}
		for _, login := range logins {
			members[login] = struct{}{}
		}
	}
	c.members = members
	c.expiresAt = c.now().Add(c.ttl)
	return nil
}

func (c *GrafanaTeamChecker) lookupTeamID(ctx context.Context, team string) (int64, error) {
	var result struct {
		Teams []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"teams"`
	}
	query := url.Values{"name": {team}, "perpage": {"1000"}}
	if err := c.getJSON(ctx, "/api/teams/search?"+query.Encode(), &result); err != nil {
		return 0, fmt.Errorf("looking up team %q: %w", team, err)
	}
	for _, candidate := range result.Teams {
		if candidate.Name == team {
			return candidate.ID, nil
		}
	}
	return 0, fmt.Errorf("team %q not found", team)
}

func (c *GrafanaTeamChecker) teamMembers(ctx context.Context, teamID int64) ([]string, error) {
	var result []struct {
		Login string `json:"login"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/api/teams/%d/members", teamID), &result); err != nil {
		return nil, fmt.Errorf("reading members of team %d: %w", teamID, err)
	}
	logins := make([]string, 0, len(result))
	for _, member := range result {
		if member.Login != "" {
			logins = append(logins, member.Login)
		}
	}
	return logins, nil
}

func (c *GrafanaTeamChecker) getJSON(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("grafana API returned %s", resp.Status)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decoding grafana API response: %w", err)
	}
	return nil
}
