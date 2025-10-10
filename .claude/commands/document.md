---
description: Create comprehensive code documentation with context integration
tags: [documentation, code-docs, technical-writing]
version: 1.0.0
---

# document

## Context-Driven Code Documentation

Before documenting any code:

1. **Read `.agent/README.md`** to understand documentation standards and structure
2. **Check `.agent/System/`** for system architecture and how this code fits in
3. **Review `.agent/SOP/`** for documentation best practices and templates
4. **Reference `.agent/Tasks/`** for feature context and acceptance criteria

## Documentation Process

Write comprehensive documentation for [code] following this systematic workflow:

### 1. Context Discovery Phase

- Understand the code's role in the system (reference `.agent/System/`)
- Identify target audience: new developers, experienced devs, or both
- Determine documentation scope: inline comments, API docs, or comprehensive guide
- Check if related documentation exists in `.agent/`

### 2. Content Planning Phase

- Define documentation structure and sections
- Identify key components to document
- Plan examples and use cases
- Determine what goes in code vs. `.agent/` docs

### 3. Overview Documentation

- **Purpose**: What problem does this code solve?
- **Scope**: What's included and what's not
- **Context**: How it fits in the larger system (reference `.agent/System/`)
- **Audience**: Who should use this code
- **Prerequisites**: What knowledge or setup is required

### 4. Component Documentation

- Break down each major component
- Explain logic and functionality
- Document parameters, return values, and side effects
- Include complexity and performance characteristics
- Reference related patterns from `.agent/SOP/`

### 5. Usage Documentation

- Provide practical examples
- Show common use cases
- Include code snippets with explanations
- Document integration points
- Reference system architecture from `.agent/System/`

### 6. Reference Documentation

- API reference (functions, classes, methods)
- Configuration options
- Environment variables
- Dependencies and version requirements
- Error codes and messages

### 7. Best Practices & Pitfalls

- Highlight best practices from `.agent/SOP/`
- Document common mistakes to avoid
- Include troubleshooting guide
- Add performance tips
- Security considerations

### 8. Maintenance Documentation

- Where to update `.agent/` docs when code changes
- How this relates to `.agent/System/` architecture
- Patterns to follow from `.agent/SOP/`
- Testing and validation approach

## Documentation Output Format

### 📚 Table of Contents

```markdown
1. Overview
2. Getting Started
3. Core Concepts
4. API Reference
5. Usage Examples
6. Best Practices
7. Troubleshooting
8. FAQ
9. Related Documentation
```

---

### 📋 Overview

#### Purpose

Clear, concise statement of what this code does and why it exists.

#### Scope

- **Included**: What this code covers
- **Excluded**: What's out of scope
- **Related**: Links to `.agent/System/` for broader context

#### System Context

How this code fits into the overall architecture (reference `.agent/System/`):

```bash
[System Component Diagram or Description]
```

#### Target Audience

- **New Developers**: What they need to know
- **Experienced Developers**: Advanced usage and internals
- **DevOps/SRE**: Deployment and operational concerns

---

### 🚀 Getting Started

#### Prerequisites

- Required knowledge (languages, frameworks, concepts)
- System requirements
- Dependencies (reference `.agent/System/` for tech stack)
- Environment setup

#### Quick Start

```[language]
// Minimal example to get started
```

#### Installation/Setup

Step-by-step instructions with commands.

---

### 🎯 Core Concepts

#### Architecture

How this code is structured (reference patterns from `.agent/SOP/`):

- High-level design
- Key components and their relationships
- Data flow diagrams
- Design patterns used

#### Key Components

##### Component 1: [Name]

**Purpose**: What it does

**Location**: File path and line references

**Responsibilities**:

- Responsibility 1
- Responsibility 2

**Related**: Links to `.agent/System/` or `.agent/SOP/`

##### Component 2: [Name]

[Continue for each major component...]

#### Data Models

Document data structures, schemas, types used.

---

### 📖 API Reference

#### Functions/Methods

##### `functionName(param1, param2)`

**Description**: Clear explanation of what this function does.

**Parameters**:

- `param1` (Type): Description
- `param2` (Type): Description

**Returns**:

- `ReturnType`: Description

**Throws/Errors**:

- `ErrorType`: When this occurs

**Example**:

```[language]
// Usage example
const result = functionName(value1, value2);
```

**Complexity**: Time and space complexity if relevant

**See Also**: Related functions or `.agent/SOP/` patterns

---

### 💡 Usage Examples

#### Example 1: [Common Use Case]

**Scenario**: When to use this

**Code**:

```[language]
// Complete, runnable example
```

**Explanation**: Step-by-step breakdown

**Output**: Expected results

---

#### Example 2: [Integration Scenario]

**Scenario**: How this integrates with other system components

**Code**:

```[language]
// Integration example referencing .agent/System/
```

---

#### Example 3: [Advanced Use Case]

**Scenario**: Complex or advanced usage

**Code**:

```[language]
// Advanced example
```

**Caveats**: Special considerations

---

### ✅ Best Practices

#### Do's ✓

- **Follow Project Patterns**: Reference `.agent/SOP/` for established patterns
- **Use Recommended Approaches**: Examples of good usage
- **Performance Tips**: How to use efficiently
- **Security Considerations**: Safe usage guidelines

#### Don'ts ✗

- **Common Mistakes**: What to avoid and why
- **Anti-Patterns**: Bad practices with explanations
- **Performance Pitfalls**: Operations that cause issues

#### Configuration Best Practices

Recommended settings and configurations.

#### Testing Best Practices

How to test code using this component.

---

### 🔧 Troubleshooting

#### Common Issues

##### Issue 1: [Problem Description]

**Symptoms**: What you'll see

**Cause**: Why this happens

**Solution**:

```[language]
// Fix or workaround
```

**Prevention**: How to avoid this

---

##### Issue 2: [Error Message]

**Symptoms**: Specific error or behavior

**Diagnosis**: How to identify the root cause

**Solution**: Step-by-step fix

**Related**: Links to `.agent/SOP/` debugging guide

---

#### Debugging Tips

- How to enable debug logging
- Key metrics to monitor
- Useful tools and commands

---

### ❓ FAQ

#### General Questions

**Q: When should I use this code?**
A: [Answer with context from `.agent/System/`]

**Q: What are the performance characteristics?**
A: [Answer with specifics]

**Q: Is this production-ready?**
A: [Answer with maturity level]

#### Technical Questions

**Q: How does this integrate with [component]?**
A: [Answer referencing `.agent/System/`]

**Q: Can I customize [behavior]?**
A: [Answer with examples]

**Q: What are the scalability limits?**
A: [Answer with numbers and considerations]

#### Operational Questions

**Q: How do I monitor this in production?**
A: [Answer with monitoring guidance]

**Q: What's the rollback procedure?**
A: [Answer with steps]

---

### 📚 Related Documentation

#### Internal Documentation

- [`.agent/System/[component].md`](.agent/System/component.md) - System architecture
- [`.agent/SOP/[topic].md`](.agent/SOP/topic.md) - Related patterns and practices
- [`.agent/Tasks/[feature].md`](.agent/Tasks/feature.md) - Feature context

#### External Resources

- Official framework/library documentation
- Relevant RFCs or specifications
- Blog posts or tutorials
- GitHub issues or discussions

---

### 🔄 Maintenance

#### Keeping Documentation Updated

**When to Update**:

- Code behavior changes
- New features added
- Bug fixes that affect usage
- Performance characteristics change

**Where to Update**:

- Inline code comments: Implementation details
- `.agent/System/`: Architectural changes
- `.agent/SOP/`: New patterns or best practices
- `.agent/Tasks/`: Feature-specific documentation
- This document: API and usage changes

#### Documentation Standards

Follow standards from `.agent/SOP/documentation.md` (if exists).

---

### 📝 Contributing

#### Adding Examples

Process for contributing new examples to documentation.

#### Improving Documentation

How to suggest or make improvements.

#### Reporting Issues

Where to report documentation errors or gaps.

---

## Documentation Principles

- **Clarity First**: Write for understanding, not brevity
- **Context-Aware**: Always reference `.agent/` docs for system context
- **Example-Rich**: Show, don't just tell
- **Multi-Level**: Serve both beginners and experts
- **Practical**: Focus on real-world usage
- **Maintainable**: Easy to keep up-to-date
- **Navigable**: Well-structured with clear sections
- **Actionable**: Enable readers to use the code immediately

## Documentation Checklist

### Content Completeness

- [ ] Purpose and scope clearly stated
- [ ] System context referenced from `.agent/System/`
- [ ] Prerequisites and dependencies listed
- [ ] All public APIs documented
- [ ] Examples for common use cases included
- [ ] Best practices from `.agent/SOP/` incorporated
- [ ] Common pitfalls documented
- [ ] Troubleshooting guide provided
- [ ] FAQ section included

### Quality Standards

- [ ] Code examples are runnable
- [ ] Technical accuracy verified
- [ ] Consistent formatting and style
- [ ] Clear and concise language
- [ ] Proper references to `.agent/` docs
- [ ] No outdated information
- [ ] Spell-checked and proofread

### Structure & Navigation

- [ ] Table of contents included
- [ ] Logical section organization
- [ ] Internal links working
- [ ] External references valid
- [ ] Easy to find information

### Maintenance

- [ ] Update process documented
- [ ] Ownership/responsibility clear
- [ ] Integrated with `.agent/` documentation structure
- [ ] Version information included

## Documentation Placement Guidelines

### Inline Comments (in code)

- Implementation details
- Complex logic explanations
- Non-obvious decisions
- TODOs and technical debt

### `.agent/System/` (system docs)

- Architecture and design
- Component relationships
- Integration points
- Tech stack decisions

### `.agent/SOP/` (standard practices)

- Reusable patterns
- Best practices
- Common procedures
- Style guides

### `.agent/Tasks/` (feature docs)

- Feature requirements
- Implementation plans
- Acceptance criteria
- Feature-specific notes

### Standalone Documentation (this output)

- Comprehensive guides
- API references
- Tutorials and examples
- User-facing documentation

## Post-Documentation Actions

After creating documentation:

1. **Update `.agent/README.md`** - Add entry in documentation index
2. **Update `.agent/System/`** - If architectural documentation added
3. **Update `.agent/SOP/`** - If new patterns or practices documented
4. **Commit Documentation** - Use conventional commit: `docs: add [description]`
5. **Review & Validate** - Ensure documentation is accurate and complete
