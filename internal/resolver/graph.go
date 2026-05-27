package resolver

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/prolm/prolm/pkg/prolfile"
)

const maxNameLen = 128

var packageNameRe = regexp.MustCompile(`^[a-z]([a-z0-9_-]*[a-z0-9])?(/[a-z]([a-z0-9_-]*[a-z0-9])?)?$`)

type dependencySpec struct {
	Name       string
	Constraint string
}

type requirement struct {
	Dependent  string
	Constraint string
}

func parseDependencySpec(spec string) (dependencySpec, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return dependencySpec{}, fmt.Errorf("dependency spec is empty")
	}
	if strings.ContainsRune(spec, 0) {
		return dependencySpec{}, fmt.Errorf("dependency spec contains null byte")
	}

	name := spec
	constraint := "*"
	if at := strings.LastIndex(spec, "@"); at >= 0 {
		name = strings.TrimSpace(spec[:at])
		constraint = strings.TrimSpace(spec[at+1:])
		if constraint == "" {
			constraint = "*"
		}
	}
	if err := validatePackageName(name); err != nil {
		return dependencySpec{}, err
	}
	if _, err := parseConstraint(name, constraint); err != nil {
		return dependencySpec{}, err
	}
	return dependencySpec{Name: name, Constraint: constraint}, nil
}

func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("package name is empty")
	}
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("package name contains null byte")
	}
	if len(name) > maxNameLen {
		return fmt.Errorf("package name %q exceeds %d characters", name, maxNameLen)
	}
	if !packageNameRe.MatchString(name) {
		return fmt.Errorf("package name %q is invalid", name)
	}
	return nil
}

func manifestRequirements(pf *prolfile.ProlFile) (map[string][]requirement, []string, error) {
	reqs := make(map[string][]requirement)
	for _, deps := range []map[string]string{pf.Dependencies, pf.DevDependencies} {
		for name, constraint := range deps {
			if err := validatePackageName(name); err != nil {
				return nil, nil, err
			}
			if _, err := parseConstraint(name, constraint); err != nil {
				return nil, nil, err
			}
			reqs[name] = append(reqs[name], requirement{Dependent: "root", Constraint: constraint})
		}
	}

	names := make([]string, 0, len(reqs))
	for name := range reqs {
		names = append(names, name)
	}
	sort.Strings(names)
	return reqs, names, nil
}

func parseDependencySpecs(specs []string) ([]dependencySpec, error) {
	parsed := make([]dependencySpec, 0, len(specs))
	for _, raw := range specs {
		spec, err := parseDependencySpec(raw)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, spec)
	}
	sort.Slice(parsed, func(i, j int) bool {
		if parsed[i].Name == parsed[j].Name {
			return parsed[i].Constraint < parsed[j].Constraint
		}
		return parsed[i].Name < parsed[j].Name
	})
	return parsed, nil
}
