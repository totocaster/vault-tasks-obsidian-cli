package vaulttasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var defaultSettings = VaultTasksSettings{
	DefaultFilter:               FilterPending,
	ExcludeFolders:              []string{},
	IncludeCancelledInCompleted: true,
	IncludeFolders:              []string{},
	OpenLocation:                LocationMain,
	PendingMode:                 PendingModeTodoAndInProgress,
	PinnedNotePaths:             []string{},
	PersistSectionFilter:        false,
	SavedSectionFilter:          nil,
	SectionSort:                 SectionSortSource,
	ShowConnectionsByDefault:    false,
	ShowSectionHeadings:         true,
	StatusMode:                  StatusModeExtended,
	TaskSort:                    TaskSortSource,
	NoteSort:                    NoteSortTitleAsc,
}

func LoadEnvironment(vaultPath string) (*Environment, error) {
	info, err := os.Stat(vaultPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", vaultPath)
	}

	vaultPath, err = filepath.Abs(vaultPath)
	if err != nil {
		return nil, err
	}

	settings, err := loadPluginSettings(vaultPath)
	if err != nil {
		return nil, err
	}

	appSettings, err := loadAppSettings(vaultPath)
	if err != nil {
		return nil, err
	}

	files, err := ScanVault(vaultPath)
	if err != nil {
		return nil, err
	}

	return &Environment{
		VaultPath: vaultPath,
		VaultName: filepath.Base(vaultPath),
		Settings:  settings,
		App:       appSettings,
		Files:     files,
	}, nil
}

func FindVaultRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	for {
		obsidianPath := filepath.Join(current, ".obsidian")
		info, err := os.Stat(obsidianPath)
		if err == nil && info.IsDir() {
			return current, nil
		}

		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}

	return "", errors.New("vault root not found")
}

func loadPluginSettings(vaultPath string) (VaultTasksSettings, error) {
	path := filepath.Join(vaultPath, ".obsidian", "plugins", "vault-tasks-view", "data.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultSettings, nil
		}
		return VaultTasksSettings{}, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return VaultTasksSettings{}, fmt.Errorf("parse plugin settings: %w", err)
	}

	return normalizeSettings(raw), nil
}

func loadAppSettings(vaultPath string) (AppSettings, error) {
	path := filepath.Join(vaultPath, ".obsidian", "app.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return AppSettings{}, nil
		}
		return AppSettings{}, err
	}

	var raw AppSettings
	if err := json.Unmarshal(data, &raw); err != nil {
		return AppSettings{}, fmt.Errorf("parse app settings: %w", err)
	}

	return raw, nil
}

func normalizeSettings(raw map[string]any) VaultTasksSettings {
	settings := defaultSettings

	settings.DefaultFilter = normalizeFilterValue(raw["defaultFilter"])
	settings.ExcludeFolders = normalizeFolderList(raw["excludeFolders"])
	settings.IncludeFolders = normalizeFolderList(raw["includeFolders"])
	settings.IncludeCancelledInCompleted = boolOrDefault(raw["includeCancelledInCompleted"], defaultSettings.IncludeCancelledInCompleted)
	settings.OpenLocation = normalizeOpenLocation(raw["openLocation"])
	settings.PendingMode = normalizePendingMode(raw["pendingMode"])
	settings.PinnedNotePaths = normalizeFolderList(raw["pinnedNotePaths"])
	settings.PersistSectionFilter = boolOrDefault(raw["persistSectionFilter"], defaultSettings.PersistSectionFilter)
	if settings.PersistSectionFilter {
		settings.SavedSectionFilter = normalizeSectionFilter(raw["savedSectionFilter"])
	} else {
		settings.SavedSectionFilter = nil
	}
	settings.SectionSort = normalizeSectionSort(raw["sectionSort"])
	settings.ShowConnectionsByDefault = boolOrDefault(raw["showConnectionsByDefault"], defaultSettings.ShowConnectionsByDefault)
	settings.ShowSectionHeadings = boolOrDefault(raw["showSectionHeadings"], defaultSettings.ShowSectionHeadings)
	settings.StatusMode = normalizeStatusMode(raw["statusMode"])
	settings.TaskSort = normalizeTaskSort(raw["taskSort"])
	settings.NoteSort = normalizeNoteSort(raw["noteSort"])

	return settings
}

func ResolveFilter(settings VaultTasksSettings, value string) (TaskFilter, error) {
	if strings.TrimSpace(value) == "" {
		return settings.DefaultFilter, nil
	}

	switch strings.TrimSpace(value) {
	case string(FilterAll):
		return FilterAll, nil
	case string(FilterPending):
		return FilterPending, nil
	case string(FilterCompleted):
		return FilterCompleted, nil
	default:
		return "", fmt.Errorf("invalid filter %q", value)
	}
}

func ResolveSectionFilter(settings VaultTasksSettings, value string) (*SectionFilter, error) {
	if strings.TrimSpace(value) == "" {
		if settings.PersistSectionFilter {
			return settings.SavedSectionFilter, nil
		}
		return nil, nil
	}

	if strings.EqualFold(strings.TrimSpace(value), "none") {
		return &SectionFilter{Kind: "none"}, nil
	}

	heading := strings.TrimSpace(value)
	if heading == "" {
		return nil, fmt.Errorf("invalid section filter %q", value)
	}

	return &SectionFilter{
		Kind:    "heading",
		Heading: heading,
	}, nil
}

func ResolveFormat(value string) (OutputFormat, error) {
	switch OutputFormat(strings.TrimSpace(value)) {
	case FormatView:
		return FormatView, nil
	case FormatSummary:
		return FormatSummary, nil
	case FormatJSON:
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid format %q", value)
	}
}

func ResolveWidth(readableLineLength bool, value string) (WidthMode, error) {
	if strings.TrimSpace(value) == "" {
		if readableLineLength {
			return WidthReadable, nil
		}
		return WidthFull, nil
	}

	switch WidthMode(strings.TrimSpace(value)) {
	case WidthReadable:
		return WidthReadable, nil
	case WidthFull:
		return WidthFull, nil
	default:
		return "", fmt.Errorf("invalid width %q", value)
	}
}

func MatchesFolderScope(path string, settings VaultTasksSettings) bool {
	normalized := normalizeFolderPath(path)
	includeMatch := len(settings.IncludeFolders) == 0
	if !includeMatch {
		for _, folder := range settings.IncludeFolders {
			if isPathInFolder(normalized, folder) {
				includeMatch = true
				break
			}
		}
	}

	if !includeMatch {
		return false
	}

	for _, folder := range settings.ExcludeFolders {
		if isPathInFolder(normalized, folder) {
			return false
		}
	}

	return true
}

func isPathInFolder(path string, folder string) bool {
	return path == folder || strings.HasPrefix(path, folder+"/")
}

func normalizeFilterValue(value any) TaskFilter {
	switch v := strings.TrimSpace(stringValue(value)); v {
	case "all":
		return FilterAll
	case "pending":
		return FilterPending
	case "completed":
		return FilterCompleted
	default:
		return defaultSettings.DefaultFilter
	}
}

func normalizeOpenLocation(value any) TaskViewLocation {
	switch stringValue(value) {
	case "main":
		return LocationMain
	case "sidebar":
		return LocationSidebar
	default:
		return defaultSettings.OpenLocation
	}
}

func normalizeStatusMode(value any) TaskStatusMode {
	switch stringValue(value) {
	case "standard":
		return StatusModeStandard
	case "extended":
		return StatusModeExtended
	default:
		return defaultSettings.StatusMode
	}
}

func normalizePendingMode(value any) PendingMode {
	switch stringValue(value) {
	case "todo-only":
		return PendingModeTodoOnly
	case "todo-and-in-progress":
		return PendingModeTodoAndInProgress
	default:
		return defaultSettings.PendingMode
	}
}

func normalizeNoteSort(value any) NoteSortMode {
	switch stringValue(value) {
	case string(NoteSortTitleAsc):
		return NoteSortTitleAsc
	case string(NoteSortTitleDesc):
		return NoteSortTitleDesc
	case string(NoteSortPathAsc):
		return NoteSortPathAsc
	case string(NoteSortPathDesc):
		return NoteSortPathDesc
	case string(NoteSortTaskCountDesc):
		return NoteSortTaskCountDesc
	case string(NoteSortTaskCountAsc):
		return NoteSortTaskCountAsc
	default:
		return defaultSettings.NoteSort
	}
}

func normalizeSectionSort(value any) SectionSortMode {
	switch stringValue(value) {
	case string(SectionSortSource):
		return SectionSortSource
	case string(SectionSortHeadingAsc):
		return SectionSortHeadingAsc
	case string(SectionSortHeadingDesc):
		return SectionSortHeadingDesc
	default:
		return defaultSettings.SectionSort
	}
}

func normalizeTaskSort(value any) TaskSortMode {
	switch stringValue(value) {
	case string(TaskSortSource):
		return TaskSortSource
	case string(TaskSortTextAsc):
		return TaskSortTextAsc
	case string(TaskSortTextDesc):
		return TaskSortTextDesc
	case string(TaskSortStatusSource):
		return TaskSortStatusSource
	default:
		return defaultSettings.TaskSort
	}
}

func normalizeSectionFilter(value any) *SectionFilter {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	kind := strings.TrimSpace(stringValue(raw["kind"]))
	switch kind {
	case "none":
		return &SectionFilter{Kind: "none"}
	case "heading":
		heading := strings.TrimSpace(stringValue(raw["heading"]))
		if heading == "" {
			return nil
		}
		return &SectionFilter{
			Kind:    "heading",
			Heading: heading,
		}
	default:
		return nil
	}
}

func normalizeFolderList(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		if direct, ok := value.([]string); ok {
			result := make([]string, 0, len(direct))
			for _, item := range direct {
				normalized := normalizeFolderPath(item)
				if normalized != "" && !contains(result, normalized) {
					result = append(result, normalized)
				}
			}
			return result
		}
		return []string{}
	}

	result := []string{}
	for _, item := range raw {
		normalized := normalizeFolderPath(stringValue(item))
		if normalized == "" || contains(result, normalized) {
			continue
		}
		result = append(result, normalized)
	}

	return result
}

func normalizeFolderPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	path = strings.TrimSuffix(path, "/")
	return path
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func boolOrDefault(value any, fallback bool) bool {
	boolValue, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolValue
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
