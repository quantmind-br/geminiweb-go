# Gotchas & Pitfalls

Things to watch out for in this codebase.

## [2025-12-21 19:34]
go build command is not available in this environment - verification must be done through code review only

_Context: When implementing subtask-3-1, could not run 'go build ./pkg/toolexec/...' for verification. Had to verify syntax and type references manually by reading dependent files (tool.go, result.go, registry.go)._

## [2025-12-21 19:58]
The 'go' command is not allowed in this environment. Commands like 'go test', 'go build', 'go vet' cannot be run for verification. Verification must be done through code review only.

_Context: When implementing subtask-5-1 to create unit tests, could not run 'go test ./pkg/toolexec/... -v' for verification. Had to verify test syntax and patterns through code review instead._
