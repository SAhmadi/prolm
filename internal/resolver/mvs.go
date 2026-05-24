package resolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/prolm/prolm/internal/registry"
)

func selectVersion(name string, reqs []requirement, versions []registry.PackageVersion) (registry.PackageVersion, error) {
	if len(versions) == 0 {
		return registry.PackageVersion{}, &registry.ErrPackageNotFound{Name: name, Registry: "registry"}
	}

	constraints := make([]*semver.Constraints, 0, len(reqs))
	for _, req := range reqs {
		c, err := parseConstraint(name, req.Constraint)
		if err != nil {
			return registry.PackageVersion{}, err
		}
		constraints = append(constraints, c)
	}

	type candidate struct {
		pv registry.PackageVersion
		v  *semver.Version
	}
	var candidates []candidate
	yankedCount := 0
	for _, pv := range versions {
		if pv.Name != "" {
			if err := validatePackageName(pv.Name); err != nil {
				return registry.PackageVersion{}, err
			}
			if pv.Name != name {
				return registry.PackageVersion{}, fmt.Errorf("registry returned package %q while resolving %q", pv.Name, name)
			}
		}
		if pv.Yanked {
			yankedCount++
			continue
		}
		v, err := normalizeVersion(pv.Version)
		if err != nil {
			return registry.PackageVersion{}, fmt.Errorf("%s registry version is invalid: %w", name, err)
		}
		pv.Name = name
		pv.Version = v.String()
		candidates = append(candidates, candidate{pv: pv, v: v})
	}
	if len(candidates) == 0 && yankedCount > 0 {
		return registry.PackageVersion{}, fmt.Errorf("all versions of %s are yanked", name)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].v.LessThan(candidates[j].v)
	})

	for _, c := range candidates {
		ok := true
		for _, constraint := range constraints {
			if !constraint.Check(c.v) {
				ok = false
				break
			}
		}
		if ok {
			return c.pv, nil
		}
	}

	return registry.PackageVersion{}, fmt.Errorf("no version of %s satisfies requirements: %s", name, formatRequirements(reqs))
}

func formatRequirements(reqs []requirement) string {
	parts := make([]string, 0, len(reqs))
	for _, req := range reqs {
		parts = append(parts, fmt.Sprintf("%s requires %s", req.Dependent, req.Constraint))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
