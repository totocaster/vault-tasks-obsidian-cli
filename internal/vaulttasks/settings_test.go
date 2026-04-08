package vaulttasks

import "testing"

func TestNormalizeSettingsWithPersistedSectionFilter(t *testing.T) {
	settings := normalizeSettings(map[string]any{
		"defaultFilter":            "all",
		"pinnedNotePaths":          []any{"Inbox.md", "./Projects/Alpha.md", "Inbox.md"},
		"persistSectionFilter":     true,
		"savedSectionFilter":       map[string]any{"kind": "heading", "heading": "Work"},
		"showConnectionsByDefault": true,
	})

	if settings.DefaultFilter != FilterAll {
		t.Fatalf("expected all, got %s", settings.DefaultFilter)
	}
	if len(settings.PinnedNotePaths) != 2 || settings.PinnedNotePaths[0] != "Inbox.md" || settings.PinnedNotePaths[1] != "Projects/Alpha.md" {
		t.Fatalf("unexpected pinned note paths: %#v", settings.PinnedNotePaths)
	}
	if settings.SavedSectionFilter == nil || settings.SavedSectionFilter.Kind != "heading" || settings.SavedSectionFilter.Heading != "Work" {
		t.Fatalf("unexpected saved section filter: %#v", settings.SavedSectionFilter)
	}
	if !settings.ShowConnectionsByDefault {
		t.Fatal("expected connections default to be true")
	}
	if settings.TaskSort != defaultSettings.TaskSort {
		t.Fatalf("expected default task sort %s, got %s", defaultSettings.TaskSort, settings.TaskSort)
	}
}

func TestMatchesFolderScope(t *testing.T) {
	settings := defaultSettings
	settings.IncludeFolders = []string{"Projects"}
	settings.ExcludeFolders = []string{"Projects/Archive"}

	if !MatchesFolderScope("Projects/Active/Task.md", settings) {
		t.Fatal("expected file inside included folder to match")
	}
	if MatchesFolderScope("Projects/Archive/Task.md", settings) {
		t.Fatal("expected excluded folder to win")
	}
	if MatchesFolderScope("Inbox/Task.md", settings) {
		t.Fatal("expected file outside include folders to fail")
	}
}

func TestResolveFilterAndSectionFilter(t *testing.T) {
	settings := defaultSettings
	settings.PersistSectionFilter = true
	settings.SavedSectionFilter = &SectionFilter{Kind: "heading", Heading: "Work"}

	filter, err := ResolveFilter(settings, "completed")
	if err != nil {
		t.Fatalf("resolve filter: %v", err)
	}
	if filter != FilterCompleted {
		t.Fatalf("expected completed filter, got %s", filter)
	}

	if _, err := ResolveFilter(settings, "weird"); err == nil {
		t.Fatal("expected invalid filter to fail")
	}

	sectionFilter, err := ResolveSectionFilter(settings, "")
	if err != nil {
		t.Fatalf("resolve saved section filter: %v", err)
	}
	if sectionFilter == nil || sectionFilter.Heading != "Work" {
		t.Fatalf("unexpected saved section filter: %#v", sectionFilter)
	}

	noneFilter, err := ResolveSectionFilter(settings, "none")
	if err != nil {
		t.Fatalf("resolve none section filter: %v", err)
	}
	if noneFilter == nil || noneFilter.Kind != "none" {
		t.Fatalf("unexpected none section filter: %#v", noneFilter)
	}
}
