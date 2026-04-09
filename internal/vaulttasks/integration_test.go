package vaulttasks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractFrontmatterFlags(t *testing.T) {
	current := "---\ndeferred-until: 2026-04-10\n---\n"
	incorrect := "---\ndeferred_until: '2026-04-11'\n---\n"
	hidden := "---\nhide-from-vault-tasks: yes\n---\n"

	if value := extractDeferredUntil(current); value == nil || *value != "2026-04-10" {
		t.Fatalf("unexpected deferred date from current key: %#v", value)
	}
	if value := extractDeferredUntil(incorrect); value != nil {
		t.Fatalf("expected incorrect key to be ignored, got %#v", value)
	}
	if !extractHiddenFromTaskList(hidden) {
		t.Fatal("expected hidden flag to be true")
	}
}

func TestParseContentExtractsTasksSectionsAndLinks(t *testing.T) {
	ref := NoteRef{
		AbsPath:  "/vault/Note.md",
		RelPath:  "Note.md",
		BaseName: "Note",
		LinkText: "Note",
	}
	content := strings.Join([]string{
		"---",
		"hide-from-vault-tasks: false",
		"---",
		"# Title",
		"## Work",
		"- [ ] First task",
		"```md",
		"- [ ] ignored",
		"```",
		"### Planning",
		"- [/] In progress task",
		"[[Target Note|alias]] and [inline](Other.md)",
	}, "\n")

	tasks, links := parseContent(ref, content)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].SectionHeading == nil || *tasks[0].SectionHeading != "Work" {
		t.Fatalf("unexpected first section heading: %#v", tasks[0].SectionHeading)
	}
	if tasks[1].SectionHeading == nil || *tasks[1].SectionHeading != "Planning" {
		t.Fatalf("unexpected second section heading: %#v", tasks[1].SectionHeading)
	}
	if tasks[1].StatusSymbol != TaskStatusInProgress {
		t.Fatalf("unexpected status symbol: %q", tasks[1].StatusSymbol)
	}
	if len(links) != 2 || links[0] != "Target Note|alias" || links[1] != "Other.md" {
		t.Fatalf("unexpected links: %#v", links)
	}
}

func TestLoadEnvironmentAndRenderShow(t *testing.T) {
	vaultPath := createTestVault(t)

	env, err := LoadEnvironment(vaultPath)
	if err != nil {
		t.Fatalf("load environment: %v", err)
	}

	sectionFilter, err := ResolveSectionFilter(env.Settings, "")
	if err != nil {
		t.Fatalf("resolve section filter: %v", err)
	}
	if sectionFilter == nil || sectionFilter.Kind != "heading" || sectionFilter.Heading != "Work" {
		t.Fatalf("expected persisted Work section filter, got %#v", sectionFilter)
	}

	width, err := ResolveWidth(env.App.ReadableLineLength, "")
	if err != nil {
		t.Fatalf("resolve width: %v", err)
	}

	snapshot, err := BuildSnapshot(env, ShowOptions{
		Filter:          env.Settings.DefaultFilter,
		SectionFilter:   sectionFilter,
		ShowConnections: true,
		Format:          FormatView,
		Width:           width,
	})
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}

	if len(snapshot.Groups) != 2 {
		t.Fatalf("expected 2 visible groups, got %d", len(snapshot.Groups))
	}
	if snapshot.Groups[0].Group.File.RelPath != "2026.md" {
		t.Fatalf("expected pinned 2026.md first, got %s", snapshot.Groups[0].Group.File.RelPath)
	}
	if snapshot.Groups[1].Group.File.RelPath != "Project.md" {
		t.Fatalf("expected Project.md second, got %s", snapshot.Groups[1].Group.File.RelPath)
	}

	rendered, err := RenderShow(snapshot, ShowOptions{Format: FormatView})
	if err != nil {
		t.Fatalf("render show: %v", err)
	}
	if !strings.Contains(rendered, "## [[2026|2026]] · 2026.md") {
		t.Fatalf("expected 2026 heading in output:\n%s", rendered)
	}
	if !strings.Contains(rendered, "### Work") {
		t.Fatalf("expected Work section in output:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Related to: [[Project]]") {
		t.Fatalf("expected related note backlink in output:\n%s", rendered)
	}
	if strings.Contains(rendered, "Skip me") {
		t.Fatalf("expected hidden note task to be excluded:\n%s", rendered)
	}
	if strings.Contains(rendered, "Deferred task") {
		t.Fatalf("expected deferred note to be excluded from pending output:\n%s", rendered)
	}
}

func TestRenderShowJSONAndAllFilter(t *testing.T) {
	vaultPath := createTestVault(t)

	env, err := LoadEnvironment(vaultPath)
	if err != nil {
		t.Fatalf("load environment: %v", err)
	}

	snapshot, err := BuildSnapshot(env, ShowOptions{
		Filter:          FilterAll,
		SectionFilter:   nil,
		ShowConnections: false,
		Format:          FormatJSON,
		Width:           WidthReadable,
	})
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}

	view, err := RenderShow(snapshot, ShowOptions{Format: FormatView})
	if err != nil {
		t.Fatalf("render view: %v", err)
	}
	if !strings.Contains(view, "Deferred until: 2099-06-10") {
		t.Fatalf("expected deferred note label in all view:\n%s", view)
	}

	jsonOutput, err := RenderShow(snapshot, ShowOptions{Format: FormatJSON})
	if err != nil {
		t.Fatalf("render json: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &decoded); err != nil {
		t.Fatalf("decode json output: %v", err)
	}
	if decoded["filter"] != string(FilterAll) {
		t.Fatalf("expected json filter %q, got %#v", FilterAll, decoded["filter"])
	}
}

func createTestVault(t *testing.T) string {
	t.Helper()

	vaultPath := t.TempDir()
	mustMkdirAll(t, filepath.Join(vaultPath, ".obsidian", "plugins", "vault-tasks-view"))

	mustWriteFile(t, filepath.Join(vaultPath, ".obsidian", "app.json"), `{"readableLineLength":true}`)
	mustWriteFile(t, filepath.Join(vaultPath, ".obsidian", "plugins", "vault-tasks-view", "data.json"), `{
  "defaultFilter": "pending",
  "excludeFolders": [],
  "includeCancelledInCompleted": true,
  "includeFolders": [],
  "openLocation": "main",
  "pendingMode": "todo-and-in-progress",
  "pinnedNotePaths": ["2026.md"],
  "persistSectionFilter": true,
  "savedSectionFilter": {"kind":"heading","heading":"Work"},
  "sectionSort": "source",
  "showConnectionsByDefault": false,
  "showSectionHeadings": true,
  "statusMode": "extended",
  "taskSort": "source",
  "noteSort": "title-asc"
}`)

	mustWriteFile(t, filepath.Join(vaultPath, "2026.md"), strings.Join([]string{
		"# 2026",
		"## Work",
		"- [ ] Core task",
		"## Personal",
		"- [ ] Personal task",
	}, "\n"))

	mustWriteFile(t, filepath.Join(vaultPath, "Project.md"), strings.Join([]string{
		"# Project",
		"Related: [[2026]]",
		"## Work",
		"- [ ] Project task",
	}, "\n"))

	mustWriteFile(t, filepath.Join(vaultPath, "Deferred.md"), strings.Join([]string{
		"---",
		"deferred-until: 2099-06-10",
		"---",
		"# Deferred",
		"## Work",
		"- [ ] Deferred task",
	}, "\n"))

	mustWriteFile(t, filepath.Join(vaultPath, "Hidden.md"), strings.Join([]string{
		"---",
		"hide-from-vault-tasks: true",
		"---",
		"# Hidden",
		"## Work",
		"- [ ] Skip me",
	}, "\n"))

	return vaultPath
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
