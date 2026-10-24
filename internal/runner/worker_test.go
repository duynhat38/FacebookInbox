package runner

import (
	"testing"

	"facebook-inbox/internal/domain"
)

func newTestWorker() *accountWorker {
	return &accountWorker{
		item: domain.AccountItem{ID: "a1", Value: "c_user=1; xs=x"},
		act:  newActivity("vi"),
	}
}

func kinds(steps []pageStep) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.kind
	}
	return out
}

func TestPageStepsClickFirst(t *testing.T) {
	w := newTestWorker()
	steps := w.pageSteps("636649692859585", "hello", "")
	got := kinds(steps)
	if len(got) != 2 || got[0] != "click" || got[1] != "text" {
		t.Fatalf("want [click text], got %v", got)
	}
	if !steps[0].task.ClickMessenger {
		t.Fatalf("click step task must have ClickMessenger=true")
	}
}

func TestPageStepsNoContentNoClick(t *testing.T) {
	w := newTestWorker()
	if steps := w.pageSteps("636649692859585", "", ""); len(steps) != 0 {
		t.Fatalf("empty page must produce no steps (no click), got %v", kinds(steps))
	}
}

func TestPageStepsDeleteOnlyRunsWithoutContent(t *testing.T) {
	w := newTestWorker()
	w.deleteThread = true
	// No text/photo/link: a ticked "delete thread" must still run on its own —
	// just the delete step, with no ad-click (nothing was sent).
	steps := w.pageSteps("636649692859585", "", "")
	got := kinds(steps)
	if len(got) != 1 || got[0] != "delete" {
		t.Fatalf("delete-only config must produce [delete], got %v", got)
	}
	if !steps[0].isDelete || !steps[0].task.DeleteThread {
		t.Fatalf("delete step must have isDelete and DeleteThread set")
	}
}

func TestPageStepsDeleteRunsAfterContent(t *testing.T) {
	w := newTestWorker()
	w.deleteThread = true
	// With content: click first, then the content step, then delete last.
	got := kinds(w.pageSteps("636649692859585", "hello", ""))
	if len(got) != 3 || got[0] != "click" || got[1] != "text" || got[2] != "delete" {
		t.Fatalf("want [click text delete], got %v", got)
	}
}
