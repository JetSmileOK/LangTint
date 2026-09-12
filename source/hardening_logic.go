package main

import "strings"

type languageNameCandidate struct {
	Name     string
	Language Language
}

func normalizeLanguageLabel(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}

// classifyAccessibleNameWithCandidates classifies a localized Windows input
// indicator string using names obtained from Windows NLS. Longest match wins,
// which prefers a locale display name such as "English (United States)" over
// the shorter primary language name "English". Equally long conflicting
// matches fail safe to UNKNOWN.
func classifyAccessibleNameWithCandidates(name string, candidates []languageNameCandidate) Language {
	n := normalizeLanguageLabel(name)
	if n == "" {
		return LanguageUnknown
	}
	bestLen := 0
	best := LanguageUnknown
	ambiguous := false
	for _, c := range candidates {
		cn := normalizeLanguageLabel(c.Name)
		if cn == "" || !strings.Contains(n, cn) {
			continue
		}
		if len(cn) > bestLen {
			bestLen = len(cn)
			best = c.Language
			ambiguous = false
			continue
		}
		if len(cn) == bestLen && best != c.Language {
			ambiguous = true
		}
	}
	if bestLen == 0 || ambiguous {
		return LanguageUnknown
	}
	return best
}

func normalizeWindowsPath(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, "/", "\\"))
	for strings.Contains(p, "\\\\") && !strings.HasPrefix(p, "\\\\") {
		p = strings.ReplaceAll(p, "\\\\", "\\")
	}
	p = strings.TrimRight(p, "\\")
	return strings.ToLower(p)
}

func isExpectedExplorerProcessPath(path, windowsDir string) bool {
	p := normalizeWindowsPath(path)
	wd := normalizeWindowsPath(windowsDir)
	if p == "" || wd == "" {
		return false
	}
	return p == wd+"\\explorer.exe"
}

type windowsSupport struct {
	Supported bool
	Reason    string
}

// Public v1.7 deliberately supports Windows 10 builds 1607 through 22H2.
// Windows 11 is rejected rather than silently relying on a taskbar compositor
// path that has not yet been validated interactively on Windows 11.
func windowsBuildSupport(major, minor, build uint32) windowsSupport {
	if major != 10 || minor != 0 {
		return windowsSupport{false, "unsupported Windows major/minor version"}
	}
	if build < 14393 {
		return windowsSupport{false, "Windows 10 build is older than 1607"}
	}
	if build >= 22000 {
		return windowsSupport{false, "Windows 11 is not validated by this release"}
	}
	return windowsSupport{true, "Windows 10 x64 supported range"}
}

var startupRetryDelaysMS = [...]uint32{250, 500, 1000, 2000, 4000, 8000, 16000}

func startupRetryTotalMS() uint32 {
	var total uint32
	for _, d := range startupRetryDelaysMS {
		total += d
	}
	return total
}
