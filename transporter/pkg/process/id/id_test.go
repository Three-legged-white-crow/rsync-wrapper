package id

import (
	"testing"
)

func TestChildren(t *testing.T) {
	curPid := Current()
	t.Log("current pid:", curPid)
	childrens, err := Children(curPid)
	if err != nil {
		t.Error("failed to get children pid:", err)
		t.FailNow()
	}

	t.Log("succeed to get childrens num:", len(childrens))
	if len(childrens) > 0 {
		for _, c := range childrens {
			t.Log("pid:", c.Pid, "name:", c.Name)
		}
	}
}
