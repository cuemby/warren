---
description: Detailed code explanation with context and architecture
tags: [documentation, explanation, learning]
version: 1.0.0
---

# explain

## Context-Driven Code Explanation

Before explaining any code:

1. **Read `.agent/README.md`** to understand the current system state
2. **Check `.agent/System/`** for architecture context and how this code fits in
3. **Review `.agent/SOP/`** for project patterns and conventions
4. **Reference `.agent/Tasks/`** for feature context if this code is part of a specific feature

## Explanation Process

Provide detailed explanation of [piece of code] following this systematic workflow:

### 1. Context Discovery Phase

- Identify where this code lives in the project structure
- Understand its role within the system architecture (reference `.agent/System/`)
- Determine its purpose: feature implementation, utility, integration, etc.
- Check if it's documented in `.agent/Tasks/` or `.agent/SOP/`

### 2. High-Level Overview

- **Purpose**: What problem does this code solve?
- **Input/Output**: What does it take in and what does it produce?
- **Scope**: Is it a function, class, module, or system component?
- **Dependencies**: What does it rely on? (check `.agent/System/` for integration points)

### 3. Detailed Walkthrough

- Break down the code line-by-line or block-by-block
- Explain logic flow and control structures
- Describe algorithms and data transformations
- Highlight key operations and their significance
- Explain error handling and edge case management

### 4. Technical Analysis

- **Architecture Pattern**: What design pattern is used? (reference `.agent/SOP/`)
- **Data Flow**: How data moves through the code
- **State Management**: How state is handled (if applicable)
- **Algorithm Complexity**: Time/space complexity where relevant
- **Technology Stack**: Libraries, frameworks, or APIs used (reference `.agent/System/`)

### 5. Best Practices & Patterns

- Identify best practices demonstrated in the code
- Reference similar patterns in `.agent/SOP/`
- Highlight good coding techniques
- Note adherence to project standards from `.agent/System/`
- Explain why certain approaches were chosen

### 6. Use Cases & Applications

- Primary use case: When and why is this code used?
- Alternative use cases or applications
- Integration points with other system components
- Real-world scenarios where this code executes

### 7. Improvements & Optimizations

- Potential performance optimizations
- Readability improvements
- Maintainability enhancements
- Scalability considerations
- Suggest if this pattern should be documented in `.agent/SOP/`

### 8. Documentation Recommendations

- Should this be documented in `.agent/System/` as a key component?
- Is this pattern worth adding to `.agent/SOP/`?
- Are there missing inline comments that would help?

## Explanation Output Format

Provide structured explanation:

### 📋 Summary

- **Purpose**: One-line description of what the code does
- **Location**: Where it fits in the system (reference `.agent/System/`)
- **Complexity**: Simple | Moderate | Complex
- **Audience Level**: Beginner-friendly | Intermediate | Advanced

### 🎯 What It Does

Clear, concise explanation of the code's primary function and purpose.

### 🔍 How It Works

#### High-Level Flow

```bash
Step 1: [Description]
Step 2: [Description]
Step 3: [Description]
```

#### Detailed Breakdown

**Section 1: [Component/Function Name]**

- Line-by-line or logical block explanation
- Why this approach is used
- What happens at each step

**Section 2: [Next Component]**

- Continue breakdown...

### 🏗️ Architecture & Patterns

- Design patterns used (with references to `.agent/SOP/`)
- How it fits in system architecture (reference `.agent/System/`)
- Dependencies and integration points
- Tech stack components involved

### 💡 Key Concepts

- Important algorithms or techniques
- Data structures used
- Complexity analysis (if relevant)
- Edge cases handled

### ✅ Best Practices Demonstrated

- What this code does well
- Patterns worth reusing
- References to project SOPs from `.agent/SOP/`

### 🚀 Use Cases

1. **Primary Use Case**: [Description]
2. **Secondary Use Cases**: [Description]
3. **Integration Points**: How other parts of the system use this

### 🔧 Possible Improvements

- **Performance**: [Suggestions]
- **Readability**: [Suggestions]
- **Maintainability**: [Suggestions]
- **Scalability**: [Suggestions]

### 📚 Related Documentation

- Relevant `.agent/System/` docs
- Related `.agent/SOP/` patterns
- Associated `.agent/Tasks/` features
- External resources or references

### 🎓 Learning Points

- Key takeaways for novice developers
- Advanced concepts for experienced developers
- Common pitfalls to avoid

## Explanation Principles

- **Multi-Level Understanding**: Cater to both beginners and experienced developers
- **Context-Aware**: Always reference system architecture and project patterns
- **Educational**: Teach concepts, don't just describe
- **Comprehensive**: Cover purpose, implementation, and implications
- **Practical**: Include real-world use cases and applications
- **Reference-Rich**: Link to `.agent/` docs for deeper context
- **Improvement-Focused**: Suggest optimizations and enhancements

## Explanation Checklist

- [ ] Code purpose clearly stated
- [ ] System context referenced from `.agent/System/`
- [ ] Line-by-line or logical breakdown provided
- [ ] Algorithms and data flow explained
- [ ] Best practices highlighted
- [ ] Use cases described
- [ ] Dependencies and tech stack noted
- [ ] Improvements suggested
- [ ] Accessible to multiple skill levels
- [ ] Related `.agent/` docs referenced
