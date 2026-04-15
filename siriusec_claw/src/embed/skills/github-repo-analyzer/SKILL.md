---
name: github-repo-analyzer
emoji: 🔍
description: Analyze GitHub repositories, extract code structure, and provide insights
homepage: https://github.com/siriusec/siriusec_claw
requires:
  bins:
    - git
    - curl
    - jq
  envs:
    - GITHUB_TOKEN: Optional GitHub token for higher API rate limits
---

# GitHub Repository Analyzer

Expert at analyzing GitHub repository structure, code patterns, and providing comprehensive insights.

## Capabilities

- **Repository Analysis**: Clone and analyze repository structure
- **Code Navigation**: Understand codebase organization and dependencies
- **Pattern Detection**: Identify code patterns, architectures, and conventions
- **Documentation Extraction**: Parse README, wiki, and code documentation
- **API Analysis**: Understand REST/GraphQL API structures
- **Dependency Analysis**: Analyze package dependencies and versions

## Usage

When analyzing a GitHub repository:
1. Clone or fetch the repository
2. Analyze directory structure and key files
3. Identify main components and their relationships
4. Extract important patterns and conventions
5. Provide actionable insights

## Tools Available

- `git clone/fetch`: Retrieve repository code
- `curl`: Access GitHub API
- `tree/ls`: Explore directory structure
- `grep`: Search for patterns
- `cat/read`: Read source files
- `jq`: Parse JSON responses

## Analysis Workflow

1. **Structure Analysis**
   - Identify project type (Go, Node.js, Python, etc.)
   - Map directory hierarchy
   - Find main entry points

2. **Code Patterns**
   - Identify architectural patterns
   - Find common utilities and helpers
   - Understand error handling approaches

3. **Documentation**
   - Extract README content
   - Find inline documentation
   - Identify API documentation

4. **Dependencies**
   - Analyze import/require statements
   - Check for external dependencies
   - Identify version constraints
