package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/prolm/prolm/internal/registry"
)

const defaultGitHubAPIBaseURL = "https://api.github.com"
const gitHubAPIUserAgent = "prolm-cli/1.x (+https://github.com/prolm/prolm)"

type gitHubRepoResolver struct {
	client  *http.Client
	apiBase string
}

func newGitHubRepoResolver() *gitHubRepoResolver {
	return &gitHubRepoResolver{
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (r *gitHubRepoResolver) Resolve(ctx context.Context, repoRef string) (registry.PackageVersion, error) {
	client := r.client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	base := strings.TrimRight(r.apiBase, "/")
	if base == "" {
		base = defaultGitHubAPIBaseURL
	}

	owner, repo, err := splitRepoRef(repoRef)
	if err != nil {
		return registry.PackageVersion{}, err
	}

	tag, version, err := r.resolveStableTag(ctx, client, base, owner, repo)
	if err != nil {
		return registry.PackageVersion{}, err
	}

	return registry.PackageVersion{
		Name:    repo,
		Version: version,
		URL:     fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", owner, repo, tag),
	}, nil
}

func splitRepoRef(repoRef string) (owner string, repo string, err error) {
	parts := splitPathParts(repoRef)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("github URL must include owner/repo")
	}
	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("github URL must include owner/repo")
	}
	return owner, repo, nil
}

func (r *gitHubRepoResolver) resolveStableTag(ctx context.Context, client *http.Client, base, owner, repo string) (tag string, version string, err error) {
	tagsURL := fmt.Sprintf("%s/repos/%s/%s/tags?per_page=100", base, owner, repo)
	tagNames, err := fetchGitHubTagNames(ctx, client, tagsURL, "tags")
	if err == nil {
		if bestTag, bestVersion := pickLatestStableSemverTag(tagNames); bestTag != "" {
			return bestTag, bestVersion, nil
		}
	}

	releasesURL := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=100", base, owner, repo)
	releaseTags, relErr := fetchGitHubReleaseTagNames(ctx, client, releasesURL)
	if relErr == nil {
		if bestTag, bestVersion := pickLatestStableSemverTag(releaseTags); bestTag != "" {
			return bestTag, bestVersion, nil
		}
	}

	if err != nil {
		return "", "", fmt.Errorf("querying github tags: %w", err)
	}
	if relErr != nil {
		return "", "", fmt.Errorf("querying github releases: %w", relErr)
	}
	return "", "", fmt.Errorf(
		"no stable semver tags/releases found for %s/%s; use an explicit tagged archive URL (for example: https://github.com/%s/%s/archive/refs/tags/vX.Y.Z.tar.gz)",
		owner, repo, owner, repo,
	)
}

func fetchGitHubTagNames(ctx context.Context, client *http.Client, endpoint, label string) ([]string, error) {
	req, err := newGitHubAPIRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, gitHubAPIError(label, resp)
	}

	var payload []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding github %s response: %w", label, err)
	}
	names := make([]string, 0, len(payload))
	for _, t := range payload {
		if strings.TrimSpace(t.Name) != "" {
			names = append(names, strings.TrimSpace(t.Name))
		}
	}
	return names, nil
}

func fetchGitHubReleaseTagNames(ctx context.Context, client *http.Client, endpoint string) ([]string, error) {
	req, err := newGitHubAPIRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, gitHubAPIError("releases", resp)
	}

	var payload []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding github releases response: %w", err)
	}
	names := make([]string, 0, len(payload))
	for _, r := range payload {
		if strings.TrimSpace(r.TagName) != "" {
			names = append(names, strings.TrimSpace(r.TagName))
		}
	}
	return names, nil
}

func pickLatestStableSemverTag(tags []string) (tag string, version string) {
	var bestRaw string
	var best *semver.Version
	for _, raw := range tags {
		trimmed := strings.TrimSpace(raw)
		normalized := strings.TrimPrefix(trimmed, "v")
		v, err := semver.NewVersion(normalized)
		if err != nil {
			continue
		}
		if v.Prerelease() != "" {
			continue
		}
		if best == nil || v.GreaterThan(best) {
			vv := *v
			best = &vv
			bestRaw = trimmed
		}
	}
	if best == nil {
		return "", ""
	}
	return bestRaw, best.String()
}

func newGitHubAPIRequest(ctx context.Context, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", gitHubAPIUserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func gitHubAPIError(label string, resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("github %s request failed", label)
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	body := strings.TrimSpace(string(bodyBytes))
	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusTooManyRequests:
		if body != "" {
			return fmt.Errorf("github %s request failed: HTTP %d (%s). Check GitHub API rate limits and set GITHUB_TOKEN", label, resp.StatusCode, body)
		}
		return fmt.Errorf("github %s request failed: HTTP %d. Check GitHub API rate limits and set GITHUB_TOKEN", label, resp.StatusCode)
	default:
		if body != "" {
			return fmt.Errorf("github %s request failed: HTTP %d (%s)", label, resp.StatusCode, body)
		}
		return fmt.Errorf("github %s request failed: HTTP %d", label, resp.StatusCode)
	}
}
