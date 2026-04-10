package skills

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

var frontmatterSep = []byte("---")

// ParseFrontmatter extracts YAML frontmatter from SKILL.md content.
// Returns parsed Metadata (may be nil if no frontmatter) and the body content.
func ParseFrontmatter(content []byte) (*Metadata, []byte) {
	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, frontmatterSep) {
		return nil, content
	}

	// Find the end of frontmatter (second ---)
	rest := trimmed[len(frontmatterSep):]
	// Skip optional newline after first ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	endIdx := bytes.Index(rest, append([]byte("\n"), frontmatterSep...))
	if endIdx < 0 {
		// Try \r\n
		endIdx = bytes.Index(rest, append([]byte("\r\n"), frontmatterSep...))
		if endIdx < 0 {
			return nil, content
		}
	}

	yamlData := rest[:endIdx]
	body := rest[endIdx:]
	// Skip the closing --- line
	nlIdx := bytes.IndexByte(body[1:], '\n')
	if nlIdx >= 0 {
		body = body[nlIdx+2:]
	} else {
		body = nil
	}

	var meta Metadata
	if err := yaml.Unmarshal(yamlData, &meta); err != nil {
		return nil, content
	}

	return &meta, body
}
