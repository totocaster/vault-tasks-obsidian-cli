package vaulttasks

import (
	"fmt"
	"sort"
	"strings"
)

func RenderShow(snapshot *Snapshot, options ShowOptions) (string, error) {
	switch options.Format {
	case FormatJSON:
		return MarshalSnapshot(snapshot)
	case FormatSummary:
		return renderSummary(snapshot), nil
	default:
		return renderView(snapshot), nil
	}
}

func RenderSections(snapshot *Snapshot) string {
	lines := []string{
		"All sections",
	}

	if snapshot.AvailableSectionFilters.HasNoSection {
		lines = append(lines, "No section")
	}
	lines = append(lines, snapshot.AvailableSectionFilters.Headings...)

	return strings.Join(lines, "\n") + "\n"
}

func RenderSettings(env *Environment) string {
	lines := []string{
		fmt.Sprintf("Vault: %s", env.VaultName),
		fmt.Sprintf("Path: %s", env.VaultPath),
		fmt.Sprintf("Default filter: %s", env.Settings.DefaultFilter),
		fmt.Sprintf("Saved section filter: %s", formatSectionFilter(env.Settings.SavedSectionFilter)),
		fmt.Sprintf("Section headings: %s", onOff(env.Settings.ShowSectionHeadings)),
		fmt.Sprintf("Connections by default: %s", onOff(env.Settings.ShowConnectionsByDefault)),
		fmt.Sprintf("Pending mode: %s", formatPendingMode(env.Settings.PendingMode)),
		fmt.Sprintf("Completed includes cancelled: %s", yesNo(env.Settings.IncludeCancelledInCompleted)),
		fmt.Sprintf("Readable line length: %s", onOff(env.App.ReadableLineLength)),
		fmt.Sprintf("Open location: %s", env.Settings.OpenLocation),
		fmt.Sprintf("Task status actions: %s", env.Settings.StatusMode),
	}

	scope := "whole vault"
	if len(env.Settings.IncludeFolders) > 0 || len(env.Settings.ExcludeFolders) > 0 {
		scope = "custom"
	}
	lines = append(lines, fmt.Sprintf("Scope: %s", scope))

	if len(env.Settings.IncludeFolders) > 0 {
		lines = append(lines, "Include folders:")
		for _, folder := range env.Settings.IncludeFolders {
			lines = append(lines, fmt.Sprintf("- %s", folder))
		}
	}
	if len(env.Settings.ExcludeFolders) > 0 {
		lines = append(lines, "Exclude folders:")
		for _, folder := range env.Settings.ExcludeFolders {
			lines = append(lines, fmt.Sprintf("- %s", folder))
		}
	}

	lines = append(lines,
		fmt.Sprintf("Note sort: %s", formatNoteSort(env.Settings.NoteSort)),
		fmt.Sprintf("Section sort: %s", formatSectionSort(env.Settings.SectionSort)),
		fmt.Sprintf("Task sort: %s", formatTaskSort(env.Settings.TaskSort)),
	)

	if len(env.Settings.PinnedNotePaths) == 0 {
		lines = append(lines, "Pinned notes: none")
	} else {
		lines = append(lines, "Pinned notes:")
		for _, path := range env.Settings.PinnedNotePaths {
			lines = append(lines, fmt.Sprintf("- %s", path))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func renderView(snapshot *Snapshot) string {
	lines := []string{
		"Vault tasks",
		fmt.Sprintf("filter: %s   section: %s   connections: %s", snapshot.Filter, formatSectionSummary(snapshot.SectionFilter), onOff(snapshot.ShowConnections)),
		"",
	}

	if len(snapshot.Groups) == 0 {
		lines = append(lines, buildEmptyStateMessage(snapshot.Filter, snapshot.SectionFilter))
		return strings.Join(lines, "\n") + "\n"
	}

	for groupIndex, visibleGroup := range snapshot.Groups {
		lines = append(lines, fmt.Sprintf("## [[%s|%s]] · %s", escapeWikiLinkText(visibleGroup.Group.File.LinkText), escapeWikiLinkText(visibleGroup.Group.NoteTitle), visibleGroup.Group.File.RelPath))

		if snapshot.Filter == FilterAll && visibleGroup.Group.DeferredUntil != nil {
			lines = append(lines, fmt.Sprintf("Deferred until: %s", *visibleGroup.Group.DeferredUntil))
		}

		if snapshot.ShowConnections {
			backlinks := snapshot.Backlinks[visibleGroup.Group.File.RelPath]
			if len(backlinks) > 0 {
				rendered := make([]string, 0, len(backlinks))
				for _, backlink := range backlinks {
					rendered = append(rendered, fmt.Sprintf("[[%s]]", escapeWikiLinkText(backlink.LinkText)))
				}
				lines = append(lines, fmt.Sprintf("Related to: %s", strings.Join(rendered, ", ")))
			}
		}

		buckets := buildSectionBuckets(visibleGroup.Tasks)
		sortSectionBuckets(buckets, snapshot.Settings.SectionSort)
		for bucketIndex := range buckets {
			sortTasks(buckets[bucketIndex].Tasks, snapshot.Settings.TaskSort)
			if snapshot.Settings.ShowSectionHeadings && buckets[bucketIndex].Heading != nil {
				lines = append(lines, fmt.Sprintf("### %s", *buckets[bucketIndex].Heading))
			}

			for _, task := range buckets[bucketIndex].Tasks {
				lines = append(lines, fmt.Sprintf("%s · %s:%d", task.RenderedLine, task.File.RelPath, task.Line+1))
			}
		}

		if groupIndex < len(snapshot.Groups)-1 {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func renderSummary(snapshot *Snapshot) string {
	totalTasks := 0
	sections := map[string]struct{}{}
	topNotes := make([]VisibleTaskGroup, len(snapshot.Groups))
	copy(topNotes, snapshot.Groups)

	for _, group := range snapshot.Groups {
		totalTasks += len(group.Tasks)
		for _, task := range group.Tasks {
			if task.SectionHeading != nil {
				sections[*task.SectionHeading] = struct{}{}
			}
		}
	}

	sort.Slice(topNotes, func(i, j int) bool {
		if len(topNotes[i].Tasks) != len(topNotes[j].Tasks) {
			return len(topNotes[i].Tasks) > len(topNotes[j].Tasks)
		}
		return topNotes[i].Group.File.RelPath < topNotes[j].Group.File.RelPath
	})

	lines := []string{
		"Vault tasks summary",
		fmt.Sprintf("Vault: %s", snapshot.VaultName),
		fmt.Sprintf("Filter: %s", snapshot.Filter),
		fmt.Sprintf("Section: %s", formatSectionSummary(snapshot.SectionFilter)),
		fmt.Sprintf("Connections: %s", onOff(snapshot.ShowConnections)),
		fmt.Sprintf("Visible notes: %d", len(snapshot.Groups)),
		fmt.Sprintf("Visible tasks: %d", totalTasks),
		fmt.Sprintf("Visible sections: %d", len(sections)),
		fmt.Sprintf("Scoped markdown files: %d", snapshot.ScopedMarkdownFiles),
		fmt.Sprintf("Task files: %d", snapshot.MarkdownTaskFiles),
		fmt.Sprintf("Hidden notes: %d", snapshot.HiddenNotesCount),
		fmt.Sprintf("Deferred notes: %d", snapshot.DeferredNotesCount),
	}

	if len(topNotes) > 0 {
		lines = append(lines, "Top notes:")
		limit := len(topNotes)
		if limit > 10 {
			limit = 10
		}
		for _, group := range topNotes[:limit] {
			lines = append(lines, fmt.Sprintf("- %s (%d) · %s", group.Group.NoteTitle, len(group.Tasks), group.Group.File.RelPath))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func buildEmptyStateMessage(filter TaskFilter, sectionFilter *SectionFilter) string {
	base := "No tasks."
	switch filter {
	case FilterPending:
		base = "No pending tasks."
	case FilterCompleted:
		base = "No completed tasks."
	}

	if sectionFilter == nil {
		return base
	}

	if sectionFilter.Kind == "none" {
		return strings.TrimSuffix(base, ".") + " without a section."
	}

	return strings.TrimSuffix(base, ".") + " in " + sectionFilter.Heading + "."
}

func formatSectionSummary(filter *SectionFilter) string {
	if filter == nil {
		return "all"
	}
	if filter.Kind == "none" {
		return "no section"
	}
	return filter.Heading
}

func formatSectionFilter(filter *SectionFilter) string {
	if filter == nil {
		return "none"
	}
	if filter.Kind == "none" {
		return "no section"
	}
	return filter.Heading
}

func formatPendingMode(mode PendingMode) string {
	switch mode {
	case PendingModeTodoOnly:
		return "[ ] only"
	default:
		return "[ ] and [/]"
	}
}

func formatNoteSort(mode NoteSortMode) string {
	switch mode {
	case NoteSortTitleDesc:
		return "title Z-A"
	case NoteSortPathAsc:
		return "path A-Z"
	case NoteSortPathDesc:
		return "path Z-A"
	case NoteSortTaskCountDesc:
		return "visible task count high-low"
	case NoteSortTaskCountAsc:
		return "visible task count low-high"
	default:
		return "title A-Z"
	}
}

func formatSectionSort(mode SectionSortMode) string {
	switch mode {
	case SectionSortHeadingAsc:
		return "heading A-Z"
	case SectionSortHeadingDesc:
		return "heading Z-A"
	default:
		return "source order"
	}
}

func formatTaskSort(mode TaskSortMode) string {
	switch mode {
	case TaskSortTextAsc:
		return "task text A-Z"
	case TaskSortTextDesc:
		return "task text Z-A"
	case TaskSortStatusSource:
		return "status, then source order"
	default:
		return "source order"
	}
}

func onOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func escapeWikiLinkText(text string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`, `|`, `\|`)
	return replacer.Replace(text)
}
