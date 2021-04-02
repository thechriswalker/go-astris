// +build noembed

package ui

import (
	"os"
)

// for dev, load from the FS everytime

// NB this loads way more than we want!
// but it's the easiest way to keep the interfaces
// identical
var Assets = os.DirFS("ui")

var Built = os.DirFS("ui/build")
