package vaulttasks

import (
	"testing"
)

func taskForTest(line int, text string, status string, section *string) TaskItem {
	return TaskItem{
		File:           NoteRef{RelPath: "Note.md", BaseName: "Note", LinkText: "Note"},
		Key:            "Note.md:1",
		Line:           line,
		RawLine:        "- [ ] " + text,
		RenderedLine:   "- [ ] " + text,
		SectionHeading: section,
		StatusSymbol:   status,
		Text:           text,
	}
}

func TestPendingModeIncludesInProgress(t *testing.T) {
	settings := defaultSettings
	settings.PendingMode = PendingModeTodoAndInProgress
	if !isPendingStatus(TaskStatusInProgress, settings) {
		t.Fatal("expected in-progress to count as pending")
	}

	settings.PendingMode = PendingModeTodoOnly
	if isPendingStatus(TaskStatusInProgress, settings) {
		t.Fatal("expected in-progress to be excluded")
	}
}

func TestCompletedModeIncludesCancelled(t *testing.T) {
	settings := defaultSettings
	settings.IncludeCancelledInCompleted = true
	if !isCompletedStatus(TaskStatusCancelled, settings) {
		t.Fatal("expected cancelled to count as completed")
	}

	settings.IncludeCancelledInCompleted = false
	if isCompletedStatus(TaskStatusCancelled, settings) {
		t.Fatal("expected cancelled to be excluded")
	}
	if !isCompletedStatus(TaskStatusDone, settings) {
		t.Fatal("expected done to count as completed")
	}
}

func TestSortVisibleGroupsPinnedFirst(t *testing.T) {
	groups := []VisibleTaskGroup{
		{
			Group: TaskGroup{File: NoteRef{RelPath: "Inbox.md", BaseName: "Inbox"}, NoteTitle: "Inbox"},
			Tasks: []TaskItem{taskForTest(1, "one", TaskStatusTodo, nil)},
		},
		{
			Group: TaskGroup{File: NoteRef{RelPath: "Areas/Alpha.md", BaseName: "Alpha"}, NoteTitle: "Alpha"},
			Tasks: []TaskItem{taskForTest(1, "one", TaskStatusTodo, nil)},
		},
		{
			Group: TaskGroup{File: NoteRef{RelPath: "Projects/Zebra.md", BaseName: "Zebra"}, NoteTitle: "Zebra"},
			Tasks: []TaskItem{taskForTest(1, "one", TaskStatusTodo, nil)},
		},
	}

	sortVisibleTaskGroups(groups, NoteSortTitleAsc, []string{"Projects/Zebra.md", "Inbox.md"})

	if groups[0].Group.NoteTitle != "Zebra" || groups[1].Group.NoteTitle != "Inbox" || groups[2].Group.NoteTitle != "Alpha" {
		t.Fatalf("unexpected order: %#v", []string{groups[0].Group.NoteTitle, groups[1].Group.NoteTitle, groups[2].Group.NoteTitle})
	}
}

func TestBuildEmptyStateMessage(t *testing.T) {
	if got := buildEmptyStateMessage(FilterPending, nil); got != "No pending tasks." {
		t.Fatalf("unexpected message %q", got)
	}
	filter := &SectionFilter{Kind: "heading", Heading: "Work"}
	if got := buildEmptyStateMessage(FilterAll, filter); got != "No tasks in Work." {
		t.Fatalf("unexpected message %q", got)
	}
}
