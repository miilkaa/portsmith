package cli

import "runtime/debug"

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return "portsmith " + info.Main.Version
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			dirty := ""
			for _, ss := range info.Settings {
				if ss.Key == "vcs.modified" && ss.Value == "true" {
					dirty = "-dirty"
					break
				}
			}
			return "portsmith (devel) " + s.Value[:7] + dirty
		}
	}
	return "portsmith (devel)"
}
