package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// varPattern matches ${VAR} and ${VAR:-default} tokens in raw YAML text.
var varPattern = regexp.MustCompile(`\$\{[^}]+\}`)

// parseDotEnv reads a .env file and returns its KEY=VALUE pairs.
// Lines starting with '#' and empty lines are skipped.
// The 'export' prefix is stripped if present.
// Values may be optionally wrapped in double or single quotes (stripped).
// If the file does not exist, an empty map is returned without error.
func parseDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("open env file %s: %w", path, err)
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and blank lines.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip optional 'export' prefix.
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		// Must contain '='.
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue // unrecognised line — skip silently
		}

		key := strings.TrimSpace(line[:idx])
		val := line[idx+1:]

		// Strip surrounding quotes (matching pair only).
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		if key != "" {
			env[key] = val
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}
	return env, nil
}

// mergeEnv builds the resolution map used during expansion.
// dotEnv values are the base; OS environment values override them for any
// key that appears in dotEnv. Keys present only in the OS environment are
// NOT included — we only check OS env for keys we know about from the file.
// This keeps the lookup set bounded and avoids enumerating all OS vars.
func mergeEnv(dotEnv map[string]string) map[string]string {
	merged := make(map[string]string, len(dotEnv))
	for k, v := range dotEnv {
		merged[k] = v
	}
	for k := range dotEnv {
		if v, ok := os.LookupEnv(k); ok {
			merged[k] = v
		}
	}
	return merged
}

// expandVars replaces all ${VAR} and ${VAR:-default} tokens in data using the
// provided env map (which should already have OS overrides applied via mergeEnv).
// For ${VAR}: if VAR is not in env, os.LookupEnv is consulted as a final
// fallback (covers vars not in .env at all). If still missing, an error is returned.
// For ${VAR:-default}: the default is used when VAR is absent from both env and OS.
// $${VAR} is not currently supported as an escape; users should avoid ${} in
// non-interpolation contexts or use literal strings.
func expandVars(data []byte, env map[string]string) ([]byte, error) {
	content := string(data)
	lines := strings.Split(content, "\n")
	var result []string
	var expandErr error

	for _, line := range lines {
		if expandErr != nil {
			break // fail-fast: stop processing once an undefined var is hit
		}

		nonComment, comment := splitYAMLComment(line)
		expanded := varPattern.ReplaceAllStringFunc(nonComment, func(match string) string {
			if expandErr != nil {
				return match // already failed — skip further expansion
			}

			inner := match[2 : len(match)-1] // strip ${ and }

			var varName, defaultVal string
			var hasDefault bool

			if idx := strings.Index(inner, ":-"); idx >= 0 {
				varName = inner[:idx]
				defaultVal = inner[idx+2:]
				hasDefault = true
			} else {
				varName = inner
			}

			// Resolution order: merged env map → OS env → default fallback
			if v, ok := env[varName]; ok {
				return v
			}
			if v, ok := os.LookupEnv(varName); ok {
				return v
			}
			if hasDefault {
				return defaultVal
			}

			expandErr = fmt.Errorf("env var %q is not set and has no default", varName)
			return match // return original if undefined & no default
		})

		result = append(result, expanded+comment)
	}

	if expandErr != nil {
		return nil, expandErr
	}

	return []byte(strings.Join(result, "\n")), nil
}

// splitYAMLComment safely separates the value from a trailing YAML comment.
// It ignores '#' characters that appear inside single or double quotes.
func splitYAMLComment(line string) (nonComment, comment string) {
	inSingle := false
	inDouble := false

	for i := 0; i < len(line); i++ {
		b := line[i]
		switch {
		case b == '\'' && !inDouble:
			inSingle = !inSingle
		case b == '"' && !inSingle:
			inDouble = !inDouble
		case b == '#' && !inSingle && !inDouble:
			return line[:i], line[i:]
		}
	}
	return line, ""
}