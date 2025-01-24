## Conditional Expressions

flow CLI uses conditional expressions to control [executable](executable.md) behavior based on runtime conditions. These expressions are written
using a simple expression language that provides access to system information, environment variables, and stored data.

### Expression Language

flow uses the [Expr](https://expr-lang.org) language for evaluating conditions. The language supports common
operators and functions while providing access to flow executable-specific context data.

**See the [Expr language documentation](https://expr-lang.org/docs/language-definition) for more information on the
expression syntax.**

#### Basic Operators

The expression language supports standard comparison and logical operators:

- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Logical: `and`, `or`, `not`
- String: `+` (concatenation), `matches` (regex matching)
- Length: `len()`

#### Available Context

When writing conditions, you have access to several context variables:

- `os`: Operating system (e.g., "linux", "darwin", "windows")
- `arch`: System architecture (e.g., "amd64", "arm64")
- `ctx`: Flow context information
    - `workspace`: Current workspace name
    - `namespace`: Current namespace
    - `workspacePath`: Path to current workspace
    - `flowFilePath`: Path to current flow file
    - `flowFileDir`: Directory containing current flow file
- `store`: Key-value map of data store contents
- `env`: Map of environment variables

### Writing Conditions

Conditions can be used in various places within flow, most commonly in the `if` field of executable configurations. Here are
some examples of common conditional patterns:

#### Operating System and Architecture Checks

Check for specific operating systems or architectures:

```yaml
executables:
  - verb: install
    name: system-specific
    serial:
      execs:
        - if: os == "darwin"
          cmd: brew install myapp
        - if: os == "linux" 
          cmd: apt-get install myapp
        - if: arch == "amd64"
          cmd: make build-amd64
        - if: arch == "arm64"
          cmd: make build-arm64
```

#### Environment Variable Checks

Make decisions based on environment variables:

```yaml
executables:
  - verb: deploy
    name: env-check
    serial:
      execs:
        - if: env["ENVIRONMENT"] == "production"
          cmd: echo "Deploying to production"
        - if: env["DEBUG"] == "true"
          cmd: echo "Debug mode enabled"
```

#### Data Store Conditions

Use stored data to control execution:

```yaml
executables:
  - verb: run
    name: data-check
    serial:
      execs:
        - cmd: flow store set feature-flag enabled
        - if: data["feature-flag"] == "enabled"
          cmd: echo "Feature is enabled"
        - if: len(data["optional-key"]) > 0
          cmd: echo "Optional key exists"
```

#### Complex Conditions

Combine multiple conditions using logical operators:

```yaml
executables:
  - verb: build
    name: complex-check
    serial:
      execs:
        - if: os == "linux" and env["CI"] == "true"
          cmd: echo "Running in Linux CI environment"
        - if: len(data["build-id"]) > 0 and (os == "darwin" or os == "linux")
          cmd: echo "Valid build on Unix-like system"
```

#### Path and Location Checks

Use context information to make path-based decisions:

```yaml
executables:
  - verb: setup
    name: path-check
    serial:
      execs:
        - if: ctx.workspace == "development"
          cmd: echo "development workspace is active"
        - if: ctx.flowFileDir matches ".*/scripts$"
          cmd: echo "In scripts directory"
```
