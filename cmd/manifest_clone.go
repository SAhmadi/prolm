package cmd

import "github.com/prolm/prolm/pkg/prolfile"

func cloneProlFile(in *prolfile.ProlFile) *prolfile.ProlFile {
	if in == nil {
		return nil
	}
	out := *in
	out.Dependencies = cloneStringMap(in.Dependencies)
	out.DevDependencies = cloneStringMap(in.DevDependencies)
	out.Runtime = cloneRuntimeMap(in.Runtime)
	out.Scripts = cloneStringMap(in.Scripts)
	out.Package.Authors = append([]string(nil), in.Package.Authors...)
	return &out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneRuntimeMap(in map[string]prolfile.RuntimeConfig) map[string]prolfile.RuntimeConfig {
	if in == nil {
		return nil
	}
	out := make(map[string]prolfile.RuntimeConfig, len(in))
	for k, v := range in {
		cfg := v
		cfg.Flags = append([]string(nil), v.Flags...)
		out[k] = cfg
	}
	return out
}
