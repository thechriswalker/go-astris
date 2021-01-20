package main

import (
	"fmt"
	"runtime"
	"strings"
)

// These variables will be linked in at build time
var (
	BuildDate string
	Commit    string
	Version   string
)

func main() {
	banner()
}

func banner() {
	width := 42 // after the :
	pad := func(s string) string {
		return s + strings.Repeat(" ", width-len(s))
	}
	fmt.Printf(`╭────────┤ Astris Voting ├───────────────────────────╮
│ Version: %s│
│  Commit: %s│
│   Built: %s│
│      OS: %s│
│    Arch: %s│
│  Author: Chris Walker <astris@thechriswalker.net>  │
╰────────────────────────────────────────────────────╯
`, pad(Version), pad(Commit[0:8]), pad(BuildDate), pad(runtime.GOOS), pad(runtime.GOARCH))
}
