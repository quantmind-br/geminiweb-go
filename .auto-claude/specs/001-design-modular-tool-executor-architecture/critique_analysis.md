# Spec Critique Analysis - Deep Dive

## CRITICAL ISSUES FOUND

### ISSUE 1: [HIGH] Scope Misalignment - TUI Integration Marked as Out of Scope

**Location**: Lines 30-32

**Current State**:
```
Out of Scope:
- Specific tool implementations (this task focuses on the architecture/framework)
- UI/TUI integration (will be handled in separate tasks)
```

**Research Says**:
- "Bubble Tea TUI Integration (CRITICAL)" (research.json line 871-874)
- "key_requirements": "Integration with Bubble Tea TUI" (line 34)
- "Tools execute within `tea.Model.Update()` message loop" (line 117)
- "Confirmation prompts = custom `tea.Msg` types" (line 118)

**Problem**: The spec treats TUI integration as out of scope, but research clearly shows it's CRITICAL and must be part of the core design. You cannot design the tool executor without understanding how confirmations work in the TUI.

**Fix Required**:
1. Remove "UI/TUI integration" from out of scope
2. Add TUI integration interfaces to the design (ConfirmationHandler, ResultRenderer)
3. Include tea.Msg patterns for tool execution in the spec

---

### ISSUE 2: [HIGH] Missing Security Architecture

**Location**: Throughout spec - security only mentioned as "middleware for validation"

**Current State**:
- Line 23: "Establish middleware/hook system for cross-cutting concerns (logging, validation, metrics)"
- No explicit security layer design
- No mention of blacklists, path validation, confirmations

**Research Says**:
- "security_model": "Multi-layered with confirmation policies, command blacklists, path validation, timeouts, and output truncation" (line 43)
- Extensive security_considerations section (lines 662-727)
- SecurityConfig with specific fields required (research phases)

**Problem**: Security is relegated to "middleware" when it should be a first-class architectural component with dedicated interfaces.

**Fix Required**:
1. Add SecurityPolicy interface to core abstractions
2. Add ConfirmationPolicy to requirements
3. Document blacklist and path validation patterns
4. Add timeout and output truncation to core design

---

### ISSUE 3: [MEDIUM] Missing Tool Call Protocol Specification

**Location**: Missing entirely from spec

**Current State**: No mention of how tools are invoked by AI

**Research Says**:
- "protocol_specification" section (lines 846-863)
- Tool call format: ```tool\n{...}\n```
- "parsing": "Use regexp to extract blocks, then json.Unmarshal to parse" (line 851)
- "streaming_integration" challenge documented (lines 859-862)

**Problem**: The spec doesn't define how tool calls arrive (JSON format, parsing, streaming considerations).

**Fix Required**:
1. Add "Tool Call Protocol" section to spec
2. Define JSON schema for tool calls
3. Mention regexp-based parsing from AI responses
4. Note streaming considerations (partial blocks)

---

### ISSUE 4: [MEDIUM] Package Location Inconsistency

**Location**: Throughout spec - uses pkg/toolexec

**Current State**: Spec says pkg/toolexec/
**Research Says**: Uses internal/tools/ in multiple places (lines 898, 903, 735)

**Analysis**: context.json confirms pkg/toolexec/ is correct. Research.json has inconsistency.

**Problem**: Minor - spec is actually correct, but should note why pkg/ vs internal/

**Fix Required**: Add note explaining pkg/toolexec is for public API (other projects can use this framework)

---

### ISSUE 5: [LOW] Missing Specific Tool Interface Methods

**Location**: Lines 91-95 - Tool interface definition

**Current State**:
```go
type Tool interface {
    Name() string
    Description() string
    Execute(context.Context, *Input) (*Output, error)
}
```

**Research Says**:
- Tool interface should include: "RequiresConfirmation(args map[string]any) bool" (line 584)
- This allows tools to declare if they need confirmation based on specific args

**Problem**: Missing method that's part of the security model

**Fix Required**: Add RequiresConfirmation() method to Tool interface

---

### ISSUE 6: [LOW] Input/Output Types Undefined

**Location**: Lines 91-95 - references *Input and *Output types

**Current State**: Interface uses *Input and *Output but these types are never defined in the spec

**Research Says**: Should use map[string]any for flexibility (line 584)

**Problem**: Spec references undefined types

**Fix Required**: Either:
1. Define Input/Output structs in spec, OR
2. Change to map[string]any as research suggests

---

## COMPLETENESS CHECK

### Missing from Spec vs Research Requirements:

1. ✅ Registry pattern - COVERED
2. ✅ Context-driven execution - COVERED
3. ✅ Errgroup concurrency - COVERED
4. ✅ Functional options - COVERED
5. ❌ Security layer design - MISSING
6. ❌ Confirmation system - MISSING
7. ❌ TUI integration patterns - MISSING (marked out of scope)
8. ❌ Tool call protocol/parsing - MISSING
9. ❌ Streaming response handling - MISSING
10. ✅ Error wrapping - COVERED

**Completeness Score: 5/10**

---

## CONSISTENCY CHECK

### Internal Consistency:
- ✅ Package names consistent (pkg/toolexec)
- ✅ Interface style consistent
- ✅ Pattern usage consistent
- ⚠️ References undefined types (Input/Output)

### Alignment with Research:
- ❌ Scope definition misaligned
- ❌ Security requirements incomplete
- ✅ Technical patterns aligned
- ✅ Dependencies aligned

---

## FEASIBILITY CHECK

✅ **Technically Feasible** - All patterns are well-established Go practices
✅ **Dependencies Available** - errgroup, mock already in project
⚠️ **Incomplete** - Missing critical components means implementation will fail to meet actual requirements

---

## RECOMMENDATIONS

### MUST FIX (Before Implementation):
1. Expand scope to include security layer and TUI integration interfaces
2. Add SecurityPolicy and ConfirmationHandler to core abstractions
3. Document tool call JSON protocol
4. Add RequiresConfirmation() to Tool interface
5. Define Input/Output types or switch to map[string]any

### SHOULD FIX:
1. Add note explaining pkg/ vs internal/ choice
2. Reference sugestao.md explicitly in spec
3. Add security requirements to functional requirements section
4. Document streaming response considerations

### NICE TO HAVE:
1. Example tool implementation for reference
2. Sequence diagram showing tool execution flow with TUI
3. Security threat model reference
