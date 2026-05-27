package scoring

import (
	"os"
	"path/filepath"
	"testing"
)

// repoChecklistDir walks up to find the repo-level checklists directory.
func repoChecklistDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		cand := filepath.Join(dir, "checklists")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("checklists dir not found")
	return ""
}
