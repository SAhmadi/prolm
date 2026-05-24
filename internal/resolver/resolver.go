package resolver

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/prolm/prolm/internal/lockfile"
	"github.com/prolm/prolm/internal/registry"
	"github.com/prolm/prolm/pkg/prolfile"
)

type resolution struct {
	registry registry.Registry

	requirements map[string][]requirement
	selected     map[string]registry.PackageVersion
	deps         map[string][]dependencySpec
	color        map[string]int
	stack        []string
}

// Resolve resolves manifest dependency intent into deterministic lock entries.
func Resolve(ctx context.Context, manifest *prolfile.ProlFile, reg registry.Registry) (*prolfile.LockFile, error) {
	if manifest == nil {
		manifest = &prolfile.ProlFile{}
	}
	reqs, roots, err := manifestRequirements(manifest)
	if err != nil {
		return nil, err
	}

	r := &resolution{
		registry:     reg,
		requirements: reqs,
		selected:     make(map[string]registry.PackageVersion),
		deps:         make(map[string][]dependencySpec),
		color:        make(map[string]int),
	}
	for _, root := range roots {
		if err := r.resolvePackage(ctx, root); err != nil {
			return nil, err
		}
	}

	lf := &prolfile.LockFile{
		Meta: prolfile.LockMeta{
			LockVersion: prolfile.CurrentLockVersion,
		},
	}
	hash, err := lockfile.ComputeProlfileHash(manifest)
	if err != nil {
		return nil, fmt.Errorf("computing prolfile hash: %w", err)
	}
	lf.Meta.ProlfileHash = hash

	names := make([]string, 0, len(r.selected))
	for name := range r.selected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pv := r.selected[name]
		url, err := reg.DownloadURL(ctx, name, pv.Version)
		if err != nil {
			return nil, fmt.Errorf("getting download URL for %s@%s: %w", name, pv.Version, err)
		}
		lf.Packages = append(lf.Packages, prolfile.LockEntry{
			Name:         name,
			Version:      pv.Version,
			Source:       "swi-pack-index",
			URL:          url,
			Checksum:     pv.Checksum,
			Dependencies: r.lockDependencyEdges(name),
		})
	}
	return lf, nil
}

func (r *resolution) resolvePackage(ctx context.Context, name string) error {
	if err := validatePackageName(name); err != nil {
		return err
	}
	switch r.color[name] {
	case 1:
		return fmt.Errorf("circular dependency detected: %s", r.cyclePath(name))
	case 2:
		// A completed package may need to be revisited when another dependent
		// adds a stricter requirement.
		r.color[name] = 0
	}

	r.color[name] = 1
	r.stack = append(r.stack, name)
	defer func() {
		r.stack = r.stack[:len(r.stack)-1]
		r.color[name] = 2
	}()

	versions, err := r.registry.Versions(ctx, name)
	if err != nil {
		return err
	}
	selected, err := selectVersion(name, r.requirements[name], versions)
	if err != nil {
		return err
	}
	r.selected[name] = selected

	deps, err := parseDependencySpecs(selected.Dependencies)
	if err != nil {
		return fmt.Errorf("%s@%s has invalid dependency metadata: %w", name, selected.Version, err)
	}
	r.deps[name] = deps

	for _, dep := range deps {
		r.requirements[dep.Name] = appendOrReplaceRequirement(r.requirements[dep.Name], requirement{
			Dependent:  name,
			Constraint: dep.Constraint,
		})
		if err := r.resolvePackage(ctx, dep.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r *resolution) cyclePath(name string) string {
	for i, entry := range r.stack {
		if entry == name {
			path := append([]string{}, r.stack[i:]...)
			path = append(path, name)
			return strings.Join(path, " -> ")
		}
	}
	path := append([]string{}, r.stack...)
	path = append(path, name)
	return strings.Join(path, " -> ")
}

func appendOrReplaceRequirement(reqs []requirement, next requirement) []requirement {
	for i := range reqs {
		if reqs[i].Dependent == next.Dependent {
			reqs[i] = next
			return reqs
		}
	}
	return append(reqs, next)
}

func (r *resolution) lockDependencyEdges(name string) []string {
	deps := r.deps[name]
	edges := make([]string, 0, len(deps))
	for _, dep := range deps {
		if selected, ok := r.selected[dep.Name]; ok {
			edges = append(edges, dep.Name+"@"+selected.Version)
		}
	}
	sort.Strings(edges)
	return edges
}
