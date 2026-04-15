package prompt

// PromptMode controls which sections are included in the assembled prompt.
type PromptMode string

const (
	// ModeFull includes all sections (default agent).
	ModeFull PromptMode = "full"
	// ModeMinimal omits Skills, Memory, Self-Update, Identity details (sub-agents).
	ModeMinimal PromptMode = "minimal"
	// ModeNone returns only the base identity line.
	ModeNone PromptMode = "none"
)

// SectionAvailability controls when a section is included.
type SectionAvailability string

const (
	// Always means the section is included in full and minimal modes.
	Always SectionAvailability = "always"
	// FullOnly means the section is only included in full mode.
	FullOnly SectionAvailability = "full_only"
	// None means the section is never included (manually controlled).
	None SectionAvailability = "none"
)

// PromptSection represents a single section of the system prompt.
type PromptSection struct {
	// Name is the unique identifier for this section.
	Name string
	// Priority controls ordering: lower numbers come first.
	Priority int
	// Content is the rendered text for this section.
	Content string
	// Availability controls in which PromptMode this section appears.
	Availability SectionAvailability
	// Enabled allows runtime toggling of individual sections.
	Enabled bool
	// CharCount tracks the character count of the content.
	CharCount int
}

// PromptBuilder assembles system prompts from registered sections.
type PromptBuilder struct {
	sections []*PromptSection
	mode     PromptMode
}

// NewPromptBuilder creates a builder with the given mode.
func NewPromptBuilder(mode PromptMode) *PromptBuilder {
	return &PromptBuilder{
		sections: make([]*PromptSection, 0),
		mode:     mode,
	}
}

// Register adds a section to the builder.
func (b *PromptBuilder) Register(section *PromptSection) {
	if section == nil {
		return
	}
	section.CharCount = len(section.Content)
	b.sections = append(b.sections, section)
}

// RegisterMany adds multiple sections at once.
func (b *PromptBuilder) RegisterMany(sections ...*PromptSection) {
	for _, s := range sections {
		b.Register(s)
	}
}

// Build assembles the final system prompt string.
// Sections are sorted by priority, filtered by mode and enabled status.
func (b *PromptBuilder) Build() string {
	if b.mode == ModeNone {
		return "You are a helpful AI assistant."
	}

	// Sort by priority
	sorted := make([]*PromptSection, 0, len(b.sections))
	for _, s := range b.sections {
		sorted = append(sorted, s)
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority < sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var result string
	for _, s := range sorted {
		if !s.Enabled || s.Content == "" {
			continue
		}
		if !b.shouldInclude(s) {
			continue
		}
		result += s.Content + "\n\n"
	}

	return result
}

// BuildWithDetails returns the assembled prompt plus per-section metadata.
func (b *PromptBuilder) BuildWithDetails() (prompt string, details []SectionDetail) {
	prompt = b.Build()
	details = make([]SectionDetail, 0, len(b.sections))

	for _, s := range b.sections {
		if !s.Enabled || s.Content == "" || !b.shouldInclude(s) {
			continue
		}
		details = append(details, SectionDetail{
			Name:      s.Name,
			Priority:  s.Priority,
			CharCount: s.CharCount,
			Included:  true,
		})
	}
	return
}

// SectionDetail holds metadata about a section for observability.
type SectionDetail struct {
	Name      string `json:"name"`
	Priority  int    `json:"priority"`
	CharCount int    `json:"charCount"`
	Included  bool   `json:"included"`
}

// shouldInclude determines if a section should be included based on mode.
func (b *PromptBuilder) shouldInclude(s *PromptSection) bool {
	switch s.Availability {
	case Always:
		return true
	case FullOnly:
		return b.mode == ModeFull
	case None:
		return false
	default:
		return true
	}
}

// --- Built-in Section Constructors ---

// CustomSection creates a custom section from arbitrary content.
func CustomSection(content string) *PromptSection {
	return &PromptSection{
		Name:         "custom",
		Priority:     5, // High priority, comes before identity
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// IdentitySection creates the base identity section.
func IdentitySection(agentName, description string) *PromptSection {
	content := "You are " + agentName + "."
	if description != "" {
		content += " " + description
	}
	content += "\nYou have access to tools for file operations, code editing, and system tasks. Use them when needed to help the user."
	return &PromptSection{
		Name:         "identity",
		Priority:     10,
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// ToolingSection creates the tooling guidance section.
func ToolingSection(toolNames []string) *PromptSection {
	content := "## Tooling\n\nYou have access to the following tools:\n"
	for _, name := range toolNames {
		content += "- " + name + "\n"
	}
	content += "\nWhen using tools, prefer precise operations over broad ones. Read files before editing. Verify changes after making them."
	return &PromptSection{
		Name:         "tooling",
		Priority:     20,
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// SafetySection creates the safety guardrail section.
func SafetySection() *PromptSection {
	content := "## Safety\n\n" +
		"- Do not execute commands that could harm the system or compromise security.\n" +
		"- Do not access or modify files outside the workspace without explicit user consent.\n" +
		"- Ask for confirmation before destructive operations.\n" +
		"- Report any security concerns you discover."
	return &PromptSection{
		Name:         "safety",
		Priority:     30,
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// SkillsSection creates the skills reference section (listing only, not full content).
func SkillsSection(skills []SkillRef) *PromptSection {
	if len(skills) == 0 {
		return &PromptSection{Name: "skills", Priority: 40, Content: "", Availability: FullOnly, Enabled: false}
	}

	content := "## Skills\n\nYou have access to the following skills. Use the `read` tool to load a skill's instructions when needed:\n\n"
	content += "<available_skills>\n"
	for _, s := range skills {
		content += "<skill>\n"
		content += "<name>" + s.Name + "</name>\n"
		if s.Description != "" {
			content += "<description>" + s.Description + "</description>\n"
		}
		if s.Location != "" {
			content += "<location>" + s.Location + "</location>\n"
		}
		content += "</skill>\n"
	}
	content += "</available_skills>\n"

	return &PromptSection{
		Name:         "skills",
		Priority:     40,
		Content:      content,
		Availability: FullOnly,
		Enabled:      true,
	}
}

// SkillRef is a lightweight reference to a skill (no full content).
type SkillRef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"` // file path to SKILL.md
}

// WorkspaceSection creates the workspace context section.
func WorkspaceSection(workspaceDir string) *PromptSection {
	if workspaceDir == "" {
		return &PromptSection{Name: "workspace", Priority: 50, Content: "", Availability: Always, Enabled: false}
	}
	content := "## Workspace\n\nWorking directory: " + workspaceDir
	return &PromptSection{
		Name:         "workspace",
		Priority:     50,
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// ProjectContextSection creates the bootstrap files injection section.
func ProjectContextSection(fileContents map[string]string, truncated []string) *PromptSection {
	if len(fileContents) == 0 {
		return &PromptSection{Name: "project_context", Priority: 55, Content: "", Availability: Always, Enabled: false}
	}

	content := "## Project Context\n\nThe following workspace files are provided as context:\n\n"
	for name, body := range fileContents {
		content += "### " + name + "\n" + body + "\n\n"
	}

	if len(truncated) > 0 {
		content += "> Note: The following files were truncated due to size limits: " + joinNames(truncated) + "\n"
	}

	return &PromptSection{
		Name:         "project_context",
		Priority:     55,
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// DateTimeSection creates the current date/time section.
func DateTimeSection(timezone, timeFormat string) *PromptSection {
	if timezone == "" {
		return &PromptSection{Name: "datetime", Priority: 60, Content: "", Availability: Always, Enabled: false}
	}
	content := "## Current Date & Time\n\nTimezone: " + timezone
	if timeFormat != "" {
		content += "\nTime format: " + timeFormat
	}
	return &PromptSection{
		Name:         "datetime",
		Priority:     60,
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// RuntimeSection creates the runtime environment section.
func RuntimeSection(host, os, model string) *PromptSection {
	content := "## Runtime\n"
	if host != "" {
		content += "Host: " + host + "\n"
	}
	if os != "" {
		content += "OS: " + os + "\n"
	}
	if model != "" {
		content += "Model: " + model + "\n"
	}
	if content == "## Runtime\n" {
		return &PromptSection{Name: "runtime", Priority: 70, Content: "", Availability: Always, Enabled: false}
	}
	return &PromptSection{
		Name:         "runtime",
		Priority:     70,
		Content:      content,
		Availability: Always,
		Enabled:      true,
	}
}

// MemoryRecallSection creates the memory recall section.
func MemoryRecallSection(memoryContent string) *PromptSection {
	if memoryContent == "" {
		return &PromptSection{Name: "memory_recall", Priority: 45, Content: "", Availability: FullOnly, Enabled: false}
	}
	content := "## Memory\n\n" + memoryContent
	return &PromptSection{
		Name:         "memory_recall",
		Priority:     45,
		Content:      content,
		Availability: FullOnly,
		Enabled:      true,
	}
}

func joinNames(names []string) string {
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}
