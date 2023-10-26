# Contributing to Keycloak-operator

:+1::tada: First off, thanks for taking the time to contribute! :tada::+1:

The following is a set of guidelines for contributing to Keycloak-operator, hosted in the [EPAM Organization](https://github.com/epam) on GitHub. These are mostly guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

#### Table Of Contents

[Code of Conduct](#code-of-conduct)

[How Can I Contribute?](#how-can-i-contribute)
  * [Reporting Bugs](#reporting-bugs)
  * [Suggesting Enhancements](#suggesting-enhancements)

[Styleguides](#styleguides)
  * [Git Commit Messages](#git-commit-messages)

## Code of Conduct

This project and everyone participating in it is governed by the [Keycloak-operator Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [SupportEPMD-EDP@epam.com](mailto:SupportEPMD-EDP@epam.com).

## How Can I Contribute?

### Reporting Bugs

This section guides you through submitting a bug report for Keycloak-operator. Following these guidelines helps maintainers and the community understand your report, reproduce the behavior, and find related reports.

Before creating issue, please perform a [cursory search](https://github.com/search?q=is%3Aissue+user%3Aedp-keycloak-operator) to see if the problem has already been reported. If it has **and the issue is still open**, add a comment to the existing issue instead of opening a new one.

When you are creating a bug report, please [include as many details as possible](#how-do-i-submit-a-good-bug-report). Fill out [the required template](https://github.com/epam/edp-keycloak-operator/.github/blob/master/.github/ISSUE_TEMPLATE/bug_report.md), the information it asks for helps us resolve issues faster.

> **Note:** If you find a **Closed** issue that seems like it is the same thing that you're experiencing, open a new issue and include a link to the original issue in the body of your new one.

#### How Do I Submit A (Good) Bug Report?

Bugs are tracked as [GitHub issues](https://guides.github.com/features/issues/). Create an issue on repository and provide the following information by filling in [the template](https://github.com/epam/edp-keycloak-operator/.github/blob/master/.github/ISSUE_TEMPLATE/bug_report.md).

Explain the problem and include additional details to help maintainers reproduce the problem:

* **Use a clear and descriptive title** for the issue to identify the problem.
* **Describe the exact steps which reproduce the problem** in as many details as possible. For example, start by explaining how you started Keycloak-operator, e.g. which command exactly you used in the terminal, or how you started Keycloak-operator otherwise. When listing steps, **don't just say what you did, but explain how you did it**.
* **Provide specific examples to demonstrate the steps**. Include links to files or GitHub projects, or copy/pasteable snippets, which you use in those examples. If you're providing snippets in the issue, use [Markdown code blocks](https://help.github.com/articles/markdown-basics/#multiple-lines).
* **Describe the behavior you observed after following the steps** and point out what exactly is the problem with that behavior.
* **Explain which behavior you expected to see instead and why.**
* **Include screenshots and animated GIFs** which show you following the described steps and clearly demonstrate the problem.

Include details about your configuration and environment:

* **Which version of Keycloak-operator are you using?**
* **What's the name and version of the cluster you're using**?
* **Which version of Keycloak is being integrated with?**

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion for Keycloak-operator, including completely new features and minor improvements to existing functionality. Following these guidelines helps maintainers and the community understand your suggestion and find related suggestions.

Before creating enhancement suggestions, please perform a [cursory search](https://github.com/search?q=is%3Aissue+user%3Aedp-keycloak-operator) to see if the enhancement has already been suggested. If it has, add a comment to the existing issue instead of opening a new one. When you are creating an enhancement suggestion, please [include as many details as possible](#how-do-i-submit-a-good-enhancement-suggestion). Fill in [the template](https://github.com/epam/edp-keycloak-operator/.github/blob/master/.github/ISSUE_TEMPLATE/feature_request.md), including the steps that you imagine you would take if the feature you're requesting existed.

#### How Do I Submit A (Good) Enhancement Suggestion?

Enhancement suggestions are tracked as [GitHub issues](https://guides.github.com/features/issues/). Create an issue on repository and provide the following information:

* **Use a clear and descriptive title** for the issue to identify the suggestion.
* **Provide a step-by-step description of the suggested enhancement** in as many details as possible.
* **Provide specific examples to demonstrate the steps**. Include copy/pasteable snippets which you use in those examples, as [Markdown code blocks](https://help.github.com/articles/markdown-basics/#multiple-lines).
* **Describe the current behavior** and **explain which behavior you expected to see instead** and why.
* **Include screenshots and animated GIFs** which help you demonstrate the steps or point out the part of Keycloak-operator which the suggestion is related to.
* **Explain why this enhancement would be useful**
* **Which version of Keycloak-operator are you using?**
* **What's the name and version of the cluster you're using**?
* **Which version of Keycloak is being integrated with?**

#### Local development

Keycloak-operator can be developed locally. For instructions on how to do this, see the following sections in the [Local Development](https://epam.github.io/edp-install/developer-guide/local-development/?h=local)

### Pull Requests

The process described here has several goals:

- Maintain Keycloak-operator quality
- Fix problems that are important to users
- Engage the community in working toward the best possible Keycloak-operator

Please follow these steps to have your contribution considered by the maintainers:

1. Follow all instructions in [the template](pull_request_template.md)
2. Follow the [styleguides](#styleguides)
3. After you submit your pull request, verify that all [status checks](https://help.github.com/articles/about-status-checks/) are passing <details><summary>What if the status checks are failing?</summary>If a status check is failing, and you believe that the failure is unrelated to your change, please leave a comment on the pull request explaining why you believe the failure is unrelated. A maintainer will re-run the status check for you. If we conclude that the failure was a false positive, then we will open an issue to track that problem with our status check suite.</details>

While the prerequisites above must be satisfied prior to having your pull request reviewed, the reviewer(s) may ask you to complete additional design work, tests, or other changes before your pull request can be ultimately accepted.

## Styleguides

### Pull Request

* Merge request can't contain more than one commit. Please squash your commits.

### Git Commit Messages

* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less
* Reference issues and pull requests liberally after the first line
