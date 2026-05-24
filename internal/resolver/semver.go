package resolver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var bareVersionRe = regexp.MustCompile(`^v?\d+(\.\d+)?(\.\d+)?([+-].*)?$`)

func normalizeVersion(raw string) (*semver.Version, error) {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return nil, fmt.Errorf("version is empty")
	}
	parsed, err := semver.NewVersion(v)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q: %w", raw, err)
	}
	return parsed, nil
}

func parseConstraint(pkg, raw string) (*semver.Constraints, error) {
	c := strings.TrimSpace(raw)
	if c == "" {
		return nil, fmt.Errorf("%s constraint is empty", pkg)
	}
	if c == "*" {
		parsed, err := semver.NewConstraint("*")
		if err != nil {
			return nil, fmt.Errorf("%s constraint %q is invalid: %w", pkg, raw, err)
		}
		return parsed, nil
	}
	if bareVersionRe.MatchString(c) && !strings.HasPrefix(c, "v") {
		c = "^" + c
	}
	if strings.HasPrefix(c, "v") && bareVersionRe.MatchString(c) {
		c = "^" + strings.TrimPrefix(c, "v")
	}
	parsed, err := semver.NewConstraint(c)
	if err != nil {
		return nil, fmt.Errorf("%s constraint %q is invalid: %w", pkg, raw, err)
	}
	return parsed, nil
}
