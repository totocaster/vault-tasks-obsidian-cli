package vaulttasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	taskLinePattern     = regexp.MustCompile(`^(\s*(?:[-*+]|\d+[.)])\s+\[)(.)(\].*)$`)
	taskTextPattern     = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+\[.\]\s?(.*)$`)
	headingPattern      = regexp.MustCompile(`^\s{0,3}(#{1,6})\s+(.+?)\s*$`)
	wikiLinkPattern     = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	datePattern         = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func ScanVault(vaultPath string) ([]ScannedFile, error) {
	files := []ScannedFile{}

	err := filepath.WalkDir(vaultPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == vaultPath {
			return nil
		}

		name := entry.Name()
		if entry.IsDir() {
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(name, ".") || !strings.EqualFold(filepath.Ext(name), ".md") {
			return nil
		}

		scanned, err := scanFile(vaultPath, path)
		if err != nil {
			return err
		}
		files = append(files, scanned)
		return nil
	})
	if err != nil {
		return nil, err
	}

	assignLinkTexts(files)
	return files, nil
}

func scanFile(vaultPath, absPath string) (ScannedFile, error) {
	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return ScannedFile{}, err
	}

	content := string(contentBytes)
	relPath, err := filepath.Rel(vaultPath, absPath)
	if err != nil {
		return ScannedFile{}, err
	}
	relPath = filepath.ToSlash(relPath)

	ref := NoteRef{
		AbsPath:  absPath,
		RelPath:  relPath,
		BaseName: strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)),
	}

	deferredUntil := extractDeferredUntil(content)
	hidden := extractHiddenFromTaskList(content)
	tasks, links := parseContent(ref, content)

	return ScannedFile{
		Ref:                ref,
		Content:            content,
		Tasks:              tasks,
		DeferredUntil:      deferredUntil,
		HiddenFromTaskList: hidden,
		Links:              links,
	}, nil
}

func BuildSnapshot(env *Environment, options ShowOptions) (*Snapshot, error) {
	now := time.Now()
	today := todayDate(now)
	scopedFiles := []ScannedFile{}
	groups := []TaskGroup{}
	hiddenNotesCount := 0
	deferredNotesCount := 0
	markdownTaskFiles := 0

	for _, file := range env.Files {
		if !MatchesFolderScope(file.Ref.RelPath, env.Settings) {
			continue
		}

		scopedFiles = append(scopedFiles, file)

		if file.HiddenFromTaskList {
			hiddenNotesCount++
		}
		if file.DeferredUntil != nil {
			deferredNotesCount++
		}

		if len(file.Tasks) == 0 {
			continue
		}

		markdownTaskFiles++
		group := TaskGroup{
			DeferredUntil:      file.DeferredUntil,
			File:               file.Ref,
			HiddenFromTaskList: file.HiddenFromTaskList,
			NoteTitle:          file.Ref.BaseName,
			Tasks:              cloneTasks(file.Tasks),
		}

		if len(group.Tasks) == 0 || group.HiddenFromTaskList {
			continue
		}

		sort.Slice(group.Tasks, func(i, j int) bool {
			return group.Tasks[i].Line < group.Tasks[j].Line
		})
		groups = append(groups, group)
	}

	visibleGroups := []VisibleTaskGroup{}
	for _, group := range groups {
		if options.Filter == FilterPending && isDeferred(group.DeferredUntil, today) {
			continue
		}

		visibleTasks := []TaskItem{}
		for _, task := range group.Tasks {
			if !matchesFilter(task, options.Filter, env.Settings) {
				continue
			}
			if !matchesSectionFilter(task, options.SectionFilter) {
				continue
			}
			visibleTasks = append(visibleTasks, task)
		}

		if len(visibleTasks) == 0 {
			continue
		}

		visibleGroups = append(visibleGroups, VisibleTaskGroup{
			Group: group,
			Tasks: visibleTasks,
		})
	}

	sortVisibleTaskGroups(visibleGroups, env.Settings.NoteSort, env.Settings.PinnedNotePaths)

	backlinks := buildBacklinks(scopedFiles)
	availableFilters := buildAvailableSectionFilters(groups, options.Filter, env.Settings, today)

	return &Snapshot{
		VaultPath:               env.VaultPath,
		VaultName:               env.VaultName,
		GeneratedAt:             now.Format(time.RFC3339),
		Today:                   today,
		Settings:                env.Settings,
		App:                     env.App,
		Filter:                  options.Filter,
		SectionFilter:           options.SectionFilter,
		ShowConnections:         options.ShowConnections,
		Width:                   options.Width,
		Groups:                  visibleGroups,
		AvailableSectionFilters: availableFilters,
		Backlinks:               backlinks,
		HiddenNotesCount:        hiddenNotesCount,
		DeferredNotesCount:      deferredNotesCount,
		ScopedMarkdownFiles:     len(scopedFiles),
		MarkdownTaskFiles:       markdownTaskFiles,
	}, nil
}

func buildAvailableSectionFilters(
	groups []TaskGroup,
	filter TaskFilter,
	settings VaultTasksSettings,
	today string,
) AvailableSectionFilters {
	headings := map[string]struct{}{}
	hasNoSection := false

	for _, group := range groups {
		if filter == FilterPending && isDeferred(group.DeferredUntil, today) {
			continue
		}

		for _, task := range group.Tasks {
			if !matchesFilter(task, filter, settings) {
				continue
			}
			if task.SectionHeading == nil {
				hasNoSection = true
				continue
			}
			headings[*task.SectionHeading] = struct{}{}
		}
	}

	result := make([]string, 0, len(headings))
	for heading := range headings {
		result = append(result, heading)
	}
	sort.Strings(result)

	return AvailableSectionFilters{
		HasNoSection: hasNoSection,
		Headings:     result,
	}
}

func buildBacklinks(files []ScannedFile) map[string][]NoteRef {
	suffixIndex := makeSuffixIndex(files)
	backlinks := map[string][]NoteRef{}

	for _, file := range files {
		if file.HiddenFromTaskList {
			continue
		}

		seenTargets := map[string]struct{}{}
		for _, target := range file.Links {
			relPath, ok := resolveLink(target, suffixIndex)
			if !ok || relPath == file.Ref.RelPath {
				continue
			}
			if _, exists := seenTargets[relPath]; exists {
				continue
			}
			seenTargets[relPath] = struct{}{}
			backlinks[relPath] = append(backlinks[relPath], file.Ref)
		}
	}

	for target, refs := range backlinks {
		sort.Slice(refs, func(i, j int) bool {
			left := refs[i]
			right := refs[j]
			if left.BaseName != right.BaseName {
				return left.BaseName < right.BaseName
			}
			return left.RelPath < right.RelPath
		})
		backlinks[target] = refs
	}

	return backlinks
}

type suffixEntry struct {
	relPath string
	count   int
}

func makeSuffixIndex(files []ScannedFile) map[string]suffixEntry {
	index := map[string]suffixEntry{}
	counts := map[string]int{}
	original := map[string]string{}

	for _, file := range files {
		for _, suffix := range fileSuffixes(file.Ref.RelPath) {
			key := strings.ToLower(suffix)
			counts[key]++
			if _, ok := original[key]; !ok {
				original[key] = file.Ref.RelPath
			}
		}
	}

	for _, file := range files {
		for _, suffix := range fileSuffixes(file.Ref.RelPath) {
			key := strings.ToLower(suffix)
			if _, exists := index[key]; !exists {
				index[key] = suffixEntry{
					relPath: file.Ref.RelPath,
					count:   counts[key],
				}
			}
		}
	}

	return index
}

func assignLinkTexts(files []ScannedFile) {
	suffixCounts := map[string]int{}
	for _, file := range files {
		for _, suffix := range fileSuffixes(file.Ref.RelPath) {
			suffixCounts[strings.ToLower(suffix)]++
		}
	}

	for index := range files {
		best := strings.TrimSuffix(files[index].Ref.RelPath, ".md")
		for _, suffix := range fileSuffixes(files[index].Ref.RelPath) {
			if suffixCounts[strings.ToLower(suffix)] == 1 {
				best = suffix
				break
			}
		}
		files[index].Ref.LinkText = best
		for taskIndex := range files[index].Tasks {
			files[index].Tasks[taskIndex].File.LinkText = best
		}
	}
}

func fileSuffixes(relPath string) []string {
	withoutExt := strings.TrimSuffix(relPath, ".md")
	parts := strings.Split(withoutExt, "/")
	suffixes := make([]string, 0, len(parts))
	for index := range parts {
		suffixes = append(suffixes, strings.Join(parts[index:], "/"))
	}
	return suffixes
}

func resolveLink(target string, index map[string]suffixEntry) (string, bool) {
	normalized := normalizeLinkTarget(target)
	if normalized == "" {
		return "", false
	}

	entry, ok := index[strings.ToLower(normalized)]
	if !ok || entry.count != 1 {
		return "", false
	}

	return entry.relPath, true
}

func normalizeLinkTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	target = strings.Split(target, "|")[0]
	target = strings.Split(target, "#")[0]
	target = strings.Split(target, "^")[0]
	target = strings.TrimSpace(target)
	target = strings.ReplaceAll(target, "\\", "/")
	target = strings.TrimPrefix(target, "./")
	target = strings.TrimPrefix(target, "/")
	target = strings.TrimSuffix(target, ".md")
	if strings.Contains(target, "://") {
		return ""
	}
	return target
}

func parseContent(ref NoteRef, content string) ([]TaskItem, []string) {
	lines := splitLines(content)
	tasks := []TaskItem{}
	links := []string{}

	inCodeFence := false
	frontmatterEnd := frontmatterEndLine(lines)
	var currentHeading *string
	var currentHeadingLine *int

	for lineIndex, rawLine := range lines {
		if lineIndex <= frontmatterEnd {
			continue
		}

		trimmed := strings.TrimSpace(rawLine)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeFence = !inCodeFence
			continue
		}

		if inCodeFence {
			continue
		}

		links = append(links, extractLinks(rawLine)...)

		if heading := extractHeading(rawLine); heading != nil {
			currentHeading = heading
			line := lineIndex
			currentHeadingLine = &line
			continue
		}

		match := taskLinePattern.FindStringSubmatch(rawLine)
		if match == nil {
			continue
		}

		textMatch := taskTextPattern.FindStringSubmatch(rawLine)
		if textMatch == nil {
			continue
		}

		renderedLine := strings.TrimLeftFunc(rawLine, unicode.IsSpace)
		statusSymbol := normalizeTaskStatusSymbol(match[2])
		task := TaskItem{
			File:         ref,
			Key:          fmt.Sprintf("%s:%d", ref.RelPath, lineIndex),
			Line:         lineIndex,
			RawLine:      rawLine,
			RenderedLine: renderedLine,
			StatusSymbol: statusSymbol,
			Text:         textMatch[1],
		}
		if currentHeading != nil {
			headingCopy := *currentHeading
			task.SectionHeading = &headingCopy
		}
		if currentHeadingLine != nil {
			lineCopy := *currentHeadingLine
			task.SectionLine = &lineCopy
		}

		tasks = append(tasks, task)
	}

	return tasks, dedupeStrings(links)
}

func splitLines(content string) []string {
	return strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
}

func frontmatterEndLine(lines []string) int {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return -1
	}

	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return index
		}
	}

	return -1
}

func extractHeading(line string) *string {
	match := headingPattern.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	heading := strings.TrimSpace(match[2])
	heading = strings.TrimSpace(strings.TrimRight(heading, "#"))
	if heading == "" {
		return nil
	}
	return &heading
}

func extractLinks(line string) []string {
	links := []string{}
	for _, match := range wikiLinkPattern.FindAllStringSubmatch(line, -1) {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}
	for _, match := range markdownLinkPattern.FindAllStringSubmatch(line, -1) {
		if len(match) < 2 {
			continue
		}
		target := match[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") {
			continue
		}
		links = append(links, target)
	}
	return links
}

func extractDeferredUntil(content string) *string {
	frontmatter := extractFrontmatter(content)
	if frontmatter == "" {
		return nil
	}

	value := extractFrontmatterValue(frontmatter, FrontmatterDeferredKey)
	if value == "" {
		return nil
	}

	normalized := unquote(strings.TrimSpace(value))
	if !datePattern.MatchString(normalized) {
		return nil
	}

	return &normalized
}

func extractHiddenFromTaskList(content string) bool {
	frontmatter := extractFrontmatter(content)
	if frontmatter == "" {
		return false
	}

	value := strings.ToLower(unquote(strings.TrimSpace(extractFrontmatterValue(frontmatter, FrontmatterHiddenKey))))
	switch value {
	case "true", "yes", "on":
		return true
	default:
		return false
	}
}

func extractFrontmatter(content string) string {
	lines := splitLines(content)
	endLine := frontmatterEndLine(lines)
	if endLine == -1 {
		return ""
	}
	return strings.Join(lines[1:endLine], "\n")
}

func extractFrontmatterValue(frontmatter string, key string) string {
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) == key {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if value[0] == '"' && value[len(value)-1] == '"' {
			return value[1 : len(value)-1]
		}
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func cloneTasks(tasks []TaskItem) []TaskItem {
	cloned := make([]TaskItem, len(tasks))
	copy(cloned, tasks)
	return cloned
}

func isDeferred(deferredUntil *string, today string) bool {
	return deferredUntil != nil && *deferredUntil > today
}

func matchesFilter(task TaskItem, filter TaskFilter, settings VaultTasksSettings) bool {
	switch filter {
	case FilterPending:
		return isPendingStatus(task.StatusSymbol, settings)
	case FilterCompleted:
		return isCompletedStatus(task.StatusSymbol, settings)
	default:
		return true
	}
}

func matchesSectionFilter(task TaskItem, sectionFilter *SectionFilter) bool {
	if sectionFilter == nil {
		return true
	}

	switch sectionFilter.Kind {
	case "none":
		return task.SectionHeading == nil
	case "heading":
		return task.SectionHeading != nil && *task.SectionHeading == sectionFilter.Heading
	default:
		return true
	}
}

func isPendingStatus(status string, settings VaultTasksSettings) bool {
	return status == TaskStatusTodo ||
		(settings.PendingMode == PendingModeTodoAndInProgress && status == TaskStatusInProgress)
}

func isCompletedStatus(status string, settings VaultTasksSettings) bool {
	return status == TaskStatusDone ||
		status == TaskStatusDoneUpper ||
		(settings.IncludeCancelledInCompleted && status == TaskStatusCancelled)
}

func normalizeTaskStatusSymbol(status string) string {
	switch status {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone, TaskStatusDoneUpper, TaskStatusCancelled, TaskStatusDeferred:
		return status
	default:
		if status == "" {
			return TaskStatusTodo
		}
		return status
	}
}

func sortVisibleTaskGroups(groups []VisibleTaskGroup, mode NoteSortMode, pinnedPaths []string) {
	sort.Slice(groups, func(i, j int) bool {
		return compareVisibleTaskGroups(groups[i], groups[j], mode, pinnedPaths) < 0
	})
}

func compareVisibleTaskGroups(left, right VisibleTaskGroup, mode NoteSortMode, pinnedPaths []string) int {
	leftPinned := pinnedIndex(left.Group.File.RelPath, pinnedPaths)
	rightPinned := pinnedIndex(right.Group.File.RelPath, pinnedPaths)

	if leftPinned != -1 || rightPinned != -1 {
		switch {
		case leftPinned == -1:
			return 1
		case rightPinned == -1:
			return -1
		default:
			return leftPinned - rightPinned
		}
	}

	switch mode {
	case NoteSortTitleDesc:
		return compareStrings(right.Group.NoteTitle, left.Group.NoteTitle, left.Group.File.RelPath, right.Group.File.RelPath)
	case NoteSortPathAsc:
		return compareStrings(left.Group.File.RelPath, right.Group.File.RelPath, left.Group.NoteTitle, right.Group.NoteTitle)
	case NoteSortPathDesc:
		return compareStrings(right.Group.File.RelPath, left.Group.File.RelPath, left.Group.NoteTitle, right.Group.NoteTitle)
	case NoteSortTaskCountDesc:
		if diff := len(right.Tasks) - len(left.Tasks); diff != 0 {
			return diff
		}
		return compareStrings(left.Group.NoteTitle, right.Group.NoteTitle, left.Group.File.RelPath, right.Group.File.RelPath)
	case NoteSortTaskCountAsc:
		if diff := len(left.Tasks) - len(right.Tasks); diff != 0 {
			return diff
		}
		return compareStrings(left.Group.NoteTitle, right.Group.NoteTitle, left.Group.File.RelPath, right.Group.File.RelPath)
	default:
		return compareStrings(left.Group.NoteTitle, right.Group.NoteTitle, left.Group.File.RelPath, right.Group.File.RelPath)
	}
}

func pinnedIndex(path string, pinnedPaths []string) int {
	for index, pinned := range pinnedPaths {
		if pinned == path {
			return index
		}
	}
	return -1
}

func buildSectionBuckets(tasks []TaskItem) []RenderSectionBucket {
	buckets := []RenderSectionBucket{}
	currentKey := ""
	var current *RenderSectionBucket

	for _, task := range tasks {
		key := "__none__"
		line := task.Line
		if task.SectionHeading != nil && task.SectionLine != nil {
			key = fmt.Sprintf("%d:%s", *task.SectionLine, *task.SectionHeading)
			line = *task.SectionLine
		}

		if current == nil || key != currentKey {
			currentKey = key
			bucket := RenderSectionBucket{
				Heading: task.SectionHeading,
				Line:    line,
				Tasks:   []TaskItem{},
			}
			buckets = append(buckets, bucket)
			current = &buckets[len(buckets)-1]
		}
		current.Tasks = append(current.Tasks, task)
	}

	return buckets
}

func sortSectionBuckets(buckets []RenderSectionBucket, mode SectionSortMode) {
	sort.Slice(buckets, func(i, j int) bool {
		left := buckets[i]
		right := buckets[j]

		if left.Heading == nil && right.Heading == nil {
			return left.Line < right.Line
		}
		if left.Heading == nil {
			return true
		}
		if right.Heading == nil {
			return false
		}

		switch mode {
		case SectionSortHeadingAsc:
			return compareStrings(*left.Heading, *right.Heading, fmt.Sprint(left.Line), fmt.Sprint(right.Line)) < 0
		case SectionSortHeadingDesc:
			return compareStrings(*right.Heading, *left.Heading, fmt.Sprint(left.Line), fmt.Sprint(right.Line)) < 0
		default:
			if left.Line != right.Line {
				return left.Line < right.Line
			}
			return *left.Heading < *right.Heading
		}
	})
}

func sortTasks(tasks []TaskItem, mode TaskSortMode) {
	sort.Slice(tasks, func(i, j int) bool {
		return compareTasks(tasks[i], tasks[j], mode) < 0
	})
}

func compareTasks(left, right TaskItem, mode TaskSortMode) int {
	switch mode {
	case TaskSortTextAsc:
		return compareStrings(left.Text, right.Text, fmt.Sprint(left.Line), fmt.Sprint(right.Line))
	case TaskSortTextDesc:
		return compareStrings(right.Text, left.Text, fmt.Sprint(left.Line), fmt.Sprint(right.Line))
	case TaskSortStatusSource:
		if diff := statusSortRank(left.StatusSymbol) - statusSortRank(right.StatusSymbol); diff != 0 {
			return diff
		}
		if left.Line != right.Line {
			return left.Line - right.Line
		}
		return strings.Compare(left.Text, right.Text)
	default:
		if left.Line != right.Line {
			return left.Line - right.Line
		}
		return strings.Compare(left.Text, right.Text)
	}
}

func statusSortRank(status string) int {
	switch status {
	case TaskStatusTodo:
		return 0
	case TaskStatusInProgress:
		return 1
	case TaskStatusDeferred:
		return 2
	case TaskStatusDone, TaskStatusDoneUpper:
		return 3
	case TaskStatusCancelled:
		return 4
	default:
		return 5
	}
}

func compareStrings(left, right, leftFallback, rightFallback string) int {
	if diff := strings.Compare(left, right); diff != 0 {
		return diff
	}
	return strings.Compare(leftFallback, rightFallback)
}

func dedupeStrings(values []string) []string {
	result := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func MarshalSnapshot(snapshot *Snapshot) (string, error) {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
