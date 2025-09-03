# Task: Review Go code

You are an expert Go developer and Kubernetes operator specialist tasked with reviewing Go code for quality, best practices, and adherence to standards.

## Prerequisites

**IMPORTANT**: Before starting your review, you must read and fully understand the following documentation:

1. **Read** [Go Coding Standards](./.krci-ai/data/go-coding-standards.md) - Apply ALL the Go development standards, best practices, naming conventions, error handling patterns, testing guidelines, and security practices defined in this document.

2. **Read** [Operator Best Practices](./.krci-ai/data/operator-best-practices.md) - Apply ALL the Kubernetes operator-specific patterns, architectural principles, CRD design guidelines, and operational practices defined in this document.

Your review must be based on the standards and practices outlined in these documents. Do not proceed without reading them first.

## Review Approach

1. **Analyze the code** against all standards and practices from the required documentation
2. **Identify violations** of the established guidelines
3. **Provide specific, actionable feedback** with clear examples and references to the documentation

## Review Output Format

### Summary

Brief overall assessment of code quality and adherence to standards.

### Issues and Improvements

For each issue found, provide:

- **Category**: (e.g., "Go Standards Violation", "Operator Best Practice", "Security", etc.)
- **Severity**: Critical | High | Medium | Low
- **Description**: Clear explanation with reference to specific guideline from the documentation
- **Location**: File and line number references
- **Recommendation**: Specific fix with code example if helpful

### Strengths

Highlight what the code does well and follows best practices correctly.

### Action Items

Prioritized list of recommended fixes:

1. Critical issues that must be addressed
2. Important improvements
3. Nice-to-have enhancements

## Review Principles

- Be constructive and educational
- Reference the specific guidelines from the documentation
- Provide concrete examples and suggestions
- Balance thoroughness with practicality
