package config

import (
	"reflect"
	"regexp"
	"sort"
)

// envVarPattern matches both ${NAME} and bare $NAME env-var reference
// shapes — the same two forms os.Expand itself recognises (expandEnv in
// this package, config.go, delegates to os.Expand directly).
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

// EnvRefNames scans every string field reachable from v (structs, maps,
// slices, arrays, pointers) for ${VAR}/$VAR references and returns the
// sorted, de-duplicated list of variable names referenced.
//
// Moved here verbatim (kernel/config, Phase 11 Task 2) from
// kernel/httpapi/config.go's envVarPattern/collectEnvVarNames — a
// behavior-preserving relocation, not a rewrite — so both consumers share
// the ONE scanner implementation rather than maintaining two regexes in
// parallel: kernel/httpapi/config.go's GET /api/config env_vars field
// (does a referenced variable exist, boolean only, never its value — D-05),
// and kernel/pluginhost's plugin-launch environment allowlist (D-14: a
// plugin subprocess receives the value behind a ${VAR} reference ONLY when
// this instance's own raw config actually declares that reference — the
// same scan, a different consumer of its result).
func EnvRefNames(v any) []string {
	names := map[string]struct{}{}
	collectEnvVarNames(reflect.ValueOf(v), names)

	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func collectEnvVarNames(v reflect.Value, names map[string]struct{}) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		collectEnvVarNames(v.Elem(), names)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			collectEnvVarNames(v.Field(i), names)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			collectEnvVarNames(v.MapIndex(key), names)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectEnvVarNames(v.Index(i), names)
		}
	case reflect.String:
		for _, m := range envVarPattern.FindAllStringSubmatch(v.String(), -1) {
			name := m[1]
			if name == "" {
				name = m[2]
			}
			if name != "" {
				names[name] = struct{}{}
			}
		}
	}
}
