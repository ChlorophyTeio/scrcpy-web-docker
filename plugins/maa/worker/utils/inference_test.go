package utils

import (
	"context"
	"maa-server/config"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCPUListBoundaries(t *testing.T) {
	v, e := ParseCPUList("0-3,2,5")
	if e != nil || !reflect.DeepEqual(v, []int{0, 1, 2, 3, 5}) {
		t.Fatal(v, e)
	}
	for _, s := range []string{"", "-1", "2-0", "0;echo", "5000", "1-2-3"} {
		if _, e = ParseCPUList(s); e == nil {
			t.Fatal(s)
		}
	}
}
func TestInferenceOnlyPinsTaskProcesses(t *testing.T) {
	old := config.D.HomeDir
	config.D.HomeDir = t.TempDir()
	defer func() { config.D.HomeDir = old }()
	_ = os.MkdirAll(filepath.Dir(InferencePath()), 0700)
	_ = os.WriteFile(InferencePath(), []byte(`{"mode":"performance"}`), 0600)
	cmd := MaaCommand(context.Background(), "run", "example", "--batch")
	h := InferenceCPUInfo()
	if len(h.Performance) > 0 && !strings.Contains(strings.Join(cmd.Args, " "), "taskset -c "+CPUList(h.Performance)+" maa run") {
		t.Fatal(cmd.Args)
	}
	for _, args := range [][]string{{"update", "stable"}, {"hot-update"}, {"version"}} {
		c := MaaCommand(context.Background(), args...)
		if filepath.Base(c.Path) != "maa" {
			t.Fatal("maintenance affinity unexpectedly changed")
		}
	}
	_ = os.WriteFile(InferencePath(), []byte(`{"mode":"invalid"}`), 0600)
	if filepath.Base(MaaCommand(context.Background(), "run", "example").Path) != "maa" {
		t.Fatal("invalid profile must fall back")
	}
}
