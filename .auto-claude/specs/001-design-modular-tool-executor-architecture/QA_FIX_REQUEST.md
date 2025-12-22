# QA Fix Request

**Status**: REJECTED
**Date**: 2025-12-22T08:55:00Z
**QA Session**: 2

## Critical Issues to Fix

### 1. Create security.go

**Problem**: The spec requires `pkg/toolexec/security.go` with SecurityPolicy interface and validators.

**Location**: `pkg/toolexec/security.go` (new file)

**Required Fix**: Create the file implementing:
- SecurityPolicy interface with Validate(ctx, toolName, args) error
- BlacklistValidator - blocks rm -rf /, dd, mkfs commands
- PathValidator - blocks .env, .ssh/, *.pem files
- CompositeSecurityPolicy - chains multiple validators

**Verification**: go build succeeds, tests in security_test.go pass

---

### 2. Create confirmation.go

**Problem**: The spec requires `pkg/toolexec/confirmation.go` with ConfirmationHandler interface.

**Location**: `pkg/toolexec/confirmation.go` (new file)

**Required Fix**: Create ConfirmationHandler interface with RequestConfirmation(ctx, tool, args) (bool, error)

**Verification**: go build succeeds, tests in confirmation_test.go pass

---

### 3. Create protocol.go

**Problem**: The spec requires `pkg/toolexec/protocol.go` with tool call parsing.

**Location**: `pkg/toolexec/protocol.go` (new file)

**Required Fix**:
- ToolCall struct with Name, Args, Reason fields
- ParseToolCalls(text) function using regex to extract JSON from ```tool blocks

**Verification**: go build succeeds, tests in protocol_test.go pass

---

### 4. Add RequiresConfirmation() to Tool interface

**Problem**: Tool interface is missing RequiresConfirmation method.

**Location**: `pkg/toolexec/tool.go`

**Required Fix**: Add `RequiresConfirmation(args map[string]any) bool` to Tool interface

**Verification**: go build succeeds, update all test mock tools to implement method

---

### 5. Add missing error types

**Problem**: Missing ErrUserDenied and ErrSecurityViolation sentinel errors.

**Location**: `pkg/toolexec/result.go`

**Required Fix**: Add sentinel errors and corresponding typed error structs with Is(), Unwrap(), Error() methods

**Verification**: errors.Is() works with new error types

---

### 6. Integrate security and confirmation into Executor

**Problem**: Executor missing security → confirmation → execution flow.

**Location**: `pkg/toolexec/executor.go`, `pkg/toolexec/options.go`

**Required Fix**:
1. Add securityPolicy and confirmHandler to executorConfig
2. Add WithSecurityPolicy() and WithConfirmationHandler() options
3. Update Execute() to check security policy, then request confirmation before execution

**Verification**: Tests verify the full security flow

---

## After Fixes

1. Run: go build ./pkg/toolexec/...
2. Run: go test ./pkg/toolexec/... -v
3. Run: go test ./pkg/toolexec/... -cover (verify >80%)
4. Run: go test ./pkg/toolexec/... -race
5. Commit: "fix: add security, confirmation, and protocol components (qa-requested)"
6. QA will re-run and validate
