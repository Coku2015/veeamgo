# 🧭 Development Guideline: CLI Help Text & Descriptions

## 1. Purpose

This document defines the conventions and best practices for **managing help text, descriptions, and examples** in large-scale CLI projects built with Go and [Cobra](https://github.com/spf13/cobra).

Its goal is to:

* Improve **maintainability** of help content.
* Support **multi-language (i18n)** and **documentation generation**.
* Keep command source files **clean, concise, and consistent**.

---

## 2. Design Principles

| Principle                  | Description                                                                                                           |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| **Separation of concerns** | Command logic and help text must reside in separate files/packages.                                                   |
| **Centralized management** | All CLI help text should be managed under a unified directory (e.g. `pkg/helptext/`).                                 |
| **Consistency**            | All commands follow the same field conventions (`Short`, `Long`, `Example`).                                          |
| **Scalability**            | The design must allow adding localization, markdown formatting, or external documentation without structural changes. |

---

## 3. Project Structure

Recommended directory layout for a medium-to-large CLI project:

```
veeamgo/
├── cmd/
│   ├── root.go
│   ├── backup.go
│   ├── restore.go
│   └── ...
├── pkg/
│   ├── helptext/
│   │   ├── backup.go
│   │   ├── restore.go
│   │   ├── general.go
│   │   └── ...
│   └── ...
├── docs/
│   └── development_guidelines/
│       └── cli_help.md   ← this document
└── go.mod
```

---

## 4. Help Text Definition

### 4.1. In `pkg/helptext/<command>.go`

Each file defines one or more command descriptions:

```go
package helptext

const BackupShort = "Start or manage backup jobs."

const BackupLong = `
Start or manage backup jobs with flexible options.
You can specify policies, schedules, and destinations.

Examples:
  veeamgo backup start --policy daily
  veeamgo backup status
`

const BackupExample = `
# Start a daily backup
veeamgo backup start --policy daily
`
```

---

### 4.2. In `cmd/<command>.go`

Import and assign constants:

```go
import (
    "github.com/Coku2015/veeamgo/pkg/helptext"
    "github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
    Use:     "backup",
    Short:   helptext.BackupShort,
    Long:    helptext.BackupLong,
    Example: helptext.BackupExample,
}
```

This keeps command files focused purely on functionality and flags.

---

## 5. File Naming & Organization

| Type                 | File Example                 | Purpose                                         |
| -------------------- | ---------------------------- | ----------------------------------------------- |
| Command help         | `pkg/helptext/backup.go`     | Descriptions specific to one command            |
| Shared help          | `pkg/helptext/general.go`    | Common messages like “Use --help for more info” |
| Multi-level commands | `pkg/helptext/backup_job.go` | If subcommands exist (e.g., `backup job start`) |

---

## 6. Formatting Guidelines

### ✅ Do:

* Use **plain English** or **Markdown-like indentation** (Cobra preserves formatting in `Long`).
* Keep `Short` concise (under 80 characters).
* Use **fenced code blocks** or indentation for examples.
* Start sentences with uppercase and end with punctuation.
* Prefer multiline strings with backticks (`` ` ``).

### ❌ Don’t:

* Include tab characters (`\t`) — use spaces.
* Hardcode ANSI colors (handled by Cobra or output layer).
* Mix logic (e.g., flag parsing) inside help text.

---

## 7. Internationalization (i18n) Ready

To support multiple languages in future:

* Store localized strings as Go maps or JSON files.
* Example:

  ```go
  var BackupShort = map[string]string{
      "en": "Start or manage backup jobs.",
      "zh": "启动或管理备份任务。",
  }
  ```
* Use a simple helper:

  ```go
  func Text(key string, lang string) string {
      return map[key][lang]
  }
  ```
* Or load language data via `embed` from JSON:

  ```go
  //go:embed locales/en/backup.json
  var enBackup string
  ```

---

## 8. Embedding External Help Docs (optional)

If help text grows large (e.g., multi-page docs), you can store them in markdown files and embed them using Go 1.16+ `embed`:

```go
import _ "embed"

//go:embed docs/help/backup.md
var BackupLong string
```

Benefits:

* Easier to review or translate help text.
* Can be reused to generate online docs or man pages.

---

## 9. Documentation Generation

To automatically export CLI docs:

* Use Cobra’s built-in:

  ```bash
  veeamgo gen-docs --format markdown --dir ./docs/cli/
  ```
* Or via Go code:

  ```go
  import "github.com/spf13/cobra/doc"
  doc.GenMarkdownTree(rootCmd, "./docs/cli")
  ```

This allows your help text to become synchronized CLI documentation.

---

## 10. Review Checklist

| Item                                                          | Check | Status |
| ------------------------------------------------------------- | ----- | ------ |
| Help text extracted from command files                        | ✅     |        |
| Constants named consistently (e.g. `<Cmd>Short`, `<Cmd>Long`) | ✅     |        |
| Long description formatted cleanly (no tabs)                  | ✅     |        |
| Examples properly indented                                    | ✅     |        |
| Optional i18n hooks ready                                     | ⚙️    |        |
| Ready for `cobra/doc` generation                              | ✅     |        |

---

## 11. Future Enhancements

* Implement dynamic help rendering (e.g., highlight examples in color).
* Add version-specific help sections.
* Allow contextual help (per subcommand or environment).

---

## ✅ Summary

> Keep logic and text separate.
> Keep structure consistent.
> Keep the system ready for scale, documentation, and translation.

By following this guideline:

* Command files stay concise and readable.
* Help text is centralized and easy to maintain.
* Your CLI can grow cleanly with new commands or languages.

---

