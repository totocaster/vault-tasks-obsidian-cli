package vaulttasks

import "time"

const (
	TaskStatusCancelled  = "-"
	TaskStatusDeferred   = ">"
	TaskStatusDone       = "x"
	TaskStatusDoneUpper  = "X"
	TaskStatusInProgress = "/"
	TaskStatusTodo       = " "

	FrontmatterDeferredKey = "deferred-until"
	FrontmatterHiddenKey   = "hide-from-vault-tasks"
)

type TaskFilter string

const (
	FilterAll       TaskFilter = "all"
	FilterPending   TaskFilter = "pending"
	FilterCompleted TaskFilter = "completed"
)

type TaskViewLocation string

const (
	LocationMain    TaskViewLocation = "main"
	LocationSidebar TaskViewLocation = "sidebar"
)

type TaskStatusMode string

const (
	StatusModeStandard TaskStatusMode = "standard"
	StatusModeExtended TaskStatusMode = "extended"
)

type PendingMode string

const (
	PendingModeTodoOnly          PendingMode = "todo-only"
	PendingModeTodoAndInProgress PendingMode = "todo-and-in-progress"
)

type NoteSortMode string

const (
	NoteSortTitleAsc      NoteSortMode = "title-asc"
	NoteSortTitleDesc     NoteSortMode = "title-desc"
	NoteSortPathAsc       NoteSortMode = "path-asc"
	NoteSortPathDesc      NoteSortMode = "path-desc"
	NoteSortTaskCountAsc  NoteSortMode = "task-count-asc"
	NoteSortTaskCountDesc NoteSortMode = "task-count-desc"
)

type SectionSortMode string

const (
	SectionSortSource      SectionSortMode = "source"
	SectionSortHeadingAsc  SectionSortMode = "heading-asc"
	SectionSortHeadingDesc SectionSortMode = "heading-desc"
)

type TaskSortMode string

const (
	TaskSortSource       TaskSortMode = "source"
	TaskSortTextAsc      TaskSortMode = "text-asc"
	TaskSortTextDesc     TaskSortMode = "text-desc"
	TaskSortStatusSource TaskSortMode = "status-source"
)

type OutputFormat string

const (
	FormatView    OutputFormat = "view"
	FormatSummary OutputFormat = "summary"
	FormatJSON    OutputFormat = "json"
)

type WidthMode string

const (
	WidthReadable WidthMode = "readable"
	WidthFull     WidthMode = "full"
)

type SectionFilter struct {
	Kind    string `json:"kind"`
	Heading string `json:"heading,omitempty"`
}

type VaultTasksSettings struct {
	DefaultFilter               TaskFilter       `json:"defaultFilter"`
	ExcludeFolders              []string         `json:"excludeFolders"`
	IncludeCancelledInCompleted bool             `json:"includeCancelledInCompleted"`
	IncludeFolders              []string         `json:"includeFolders"`
	OpenLocation                TaskViewLocation `json:"openLocation"`
	PendingMode                 PendingMode      `json:"pendingMode"`
	PinnedNotePaths             []string         `json:"pinnedNotePaths"`
	PersistSectionFilter        bool             `json:"persistSectionFilter"`
	SavedSectionFilter          *SectionFilter   `json:"savedSectionFilter"`
	SectionSort                 SectionSortMode  `json:"sectionSort"`
	ShowConnectionsByDefault    bool             `json:"showConnectionsByDefault"`
	ShowSectionHeadings         bool             `json:"showSectionHeadings"`
	StatusMode                  TaskStatusMode   `json:"statusMode"`
	TaskSort                    TaskSortMode     `json:"taskSort"`
	NoteSort                    NoteSortMode     `json:"noteSort"`
}

type AppSettings struct {
	ReadableLineLength bool `json:"readableLineLength"`
}

type NoteRef struct {
	AbsPath  string `json:"absPath"`
	RelPath  string `json:"relPath"`
	BaseName string `json:"baseName"`
	LinkText string `json:"linkText"`
}

type TaskItem struct {
	File           NoteRef `json:"file"`
	Key            string  `json:"key"`
	Line           int     `json:"line"`
	RawLine        string  `json:"rawLine"`
	RenderedLine   string  `json:"renderedLine"`
	SectionHeading *string `json:"sectionHeading,omitempty"`
	SectionLine    *int    `json:"sectionLine,omitempty"`
	StatusSymbol   string  `json:"statusSymbol"`
	Text           string  `json:"text"`
}

type TaskGroup struct {
	DeferredUntil      *string    `json:"deferredUntil,omitempty"`
	File               NoteRef    `json:"file"`
	HiddenFromTaskList bool       `json:"hiddenFromTaskList"`
	NoteTitle          string     `json:"noteTitle"`
	Tasks              []TaskItem `json:"tasks"`
}

type VisibleTaskGroup struct {
	Group TaskGroup  `json:"group"`
	Tasks []TaskItem `json:"tasks"`
}

type RenderSectionBucket struct {
	Heading *string    `json:"heading,omitempty"`
	Line    int        `json:"line"`
	Tasks   []TaskItem `json:"tasks"`
}

type AvailableSectionFilters struct {
	HasNoSection bool     `json:"hasNoSection"`
	Headings     []string `json:"headings"`
}

type Environment struct {
	VaultPath string             `json:"vaultPath"`
	VaultName string             `json:"vaultName"`
	Settings  VaultTasksSettings `json:"settings"`
	App       AppSettings        `json:"app"`
	Files     []ScannedFile      `json:"-"`
}

type ScannedFile struct {
	Ref                NoteRef
	Content            string
	Tasks              []TaskItem
	DeferredUntil      *string
	HiddenFromTaskList bool
	Links              []string
}

type ShowOptions struct {
	Filter          TaskFilter
	SectionFilter   *SectionFilter
	ShowConnections bool
	Format          OutputFormat
	Width           WidthMode
}

type Snapshot struct {
	VaultPath               string                  `json:"vaultPath"`
	VaultName               string                  `json:"vaultName"`
	GeneratedAt             string                  `json:"generatedAt"`
	Today                   string                  `json:"today"`
	Settings                VaultTasksSettings      `json:"settings"`
	App                     AppSettings             `json:"app"`
	Filter                  TaskFilter              `json:"filter"`
	SectionFilter           *SectionFilter          `json:"sectionFilter,omitempty"`
	ShowConnections         bool                    `json:"showConnections"`
	Width                   WidthMode               `json:"width"`
	Groups                  []VisibleTaskGroup      `json:"groups"`
	AvailableSectionFilters AvailableSectionFilters `json:"availableSectionFilters"`
	Backlinks               map[string][]NoteRef    `json:"backlinks"`
	HiddenNotesCount        int                     `json:"hiddenNotesCount"`
	DeferredNotesCount      int                     `json:"deferredNotesCount"`
	ScopedMarkdownFiles     int                     `json:"scopedMarkdownFiles"`
	MarkdownTaskFiles       int                     `json:"markdownTaskFiles"`
}

func todayDate(now time.Time) string {
	return now.Format("2006-01-02")
}
