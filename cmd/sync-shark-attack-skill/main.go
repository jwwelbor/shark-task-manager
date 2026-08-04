// Command sync-shark-attack-skill restores the authored Shark Attack skill
// mirror from the canonical embedded source tree.
package main

import (
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
)

const (
	embeddedSource      = "internal/sharkdata/default_data/skills/shark-attack"
	authoredDestination = "skills/shark-attack"
)

func main() {
	if err := sharkdata.SyncSharkAttackTree(embeddedSource, authoredDestination); err != nil {
		fmt.Fprintf(os.Stderr, "sync Shark Attack skill mirror: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Synced %s -> %s\n", embeddedSource, authoredDestination)
}
