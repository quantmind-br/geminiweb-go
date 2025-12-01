╭────────────────────────────────────────────────────────────────────────────╮
│                                                                            │
│    Refactoring/Design Plan: Output Brute Content Mode for STDIN/File       │
│   Output                                                                   │
│                                                                            │
│   ## 1. Executive Summary & Goals                                          │
│                                                                            │
│   The user requires that when the application output is directed to        │
│   stdout  via  stdin  piping ( cat file | geminiweb ) or via the           │
│   output flag ( -o file.md ), the content must be raw (unrendered) and     │
│   devoid of decorative details. This means disabling the beautiful         │
│   Markdown rendering ( glamour ), the surrounding UI elements (            │
│   lipgloss  bubbles, labels), and any other decorative text (e.g.,         │
│   success messages) for these specific output modes, while ensuring        │
│   they still function normally for the standard terminal output case.      │
│                                                                            │
│   ### Key Goals:                                                           │
│                                                                            │
│   1. Introduce Raw Output Mode: Implement a flag/variable to control       │
│   the output format, defaulting to raw for non-interactive output          │
│   streams (e.g.,  -o  flag,  stdin  pipe).                                 │
│   2. Bypass Markdown & Decoration: For raw mode, the response text         │
│   must be printed directly to the relevant output ( stdout  or file)       │
│   without Markdown rendering,  lipgloss  styling, or surrounding           │
│   TUI/CLI decorative elements (labels, bubbles, thoughts, status           │
│   messages).                                                               │
│   3. Refactor  internal/commands/query.go : Isolate the rendering          │
│   logic in  runQuery  to respect the raw mode requirement.                 │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 2. Current Situation Analysis                                         │
│                                                                            │
│   The core logic for handling single queries and output is located in      │
│   internal/commands/query.go , specifically within the  runQuery           │
│   function.                                                                │
│                                                                            │
│   * Query Execution ( internal/commands/root.go ): The  rootCmd            │
│   handles input from positional arguments,  -f  flag, or  stdin , and      │
│   then calls  runQuery(prompt) .                                           │
│   * Output Handling ( internal/commands/query.go:runQuery ):               │
│     * Currently, the entire process—including connection spin-up/down,     │
│     response generation, and output rendering—is heavily coupled with      │
│     os.Stderr  and styled output ( lipgloss ).                             │
│     * The Markdown rendering happens via  render.MarkdownWithWidth         │
│     and the result is wrapped in  assistantBubbleStyle  and printed        │
│     along with  assistantLabelStyle  and  thoughtsStyle .                  │
│     * If the  -o outputFlag  is present, the raw response text is          │
│     saved, but the styled output (labels, thoughts, bubbles) is still      │
│     printed to  os.Stdout / os.Stderr  unless specifically suppressed.     │
│     The user request implies that only the raw content should be           │
│     visible on the dedicated output stream/file, and all other             │
│     decorative content should be suppressed or directed elsewhere.         │
│                                                                            │
│                                                                            │
│   Key Pain Points:                                                         │
│                                                                            │
│   1. Mixed Output Streams: Decorative/status output is mixed on  os.       │
│   Stderr  ( spin.start() ,  spin.stopWithSuccess() , clipboard             │
│   message) and the main response is printed to  os.Stdout  ( fmt.          │
│   Println(label) ,  fmt.Println(bubble) ). This mix is undesirable         │
│   when piping or redirecting the main content.                             │
│   2. No  Raw  Output Switch: There is no easy mechanism to switch off      │
│   all styling and decorative elements to get a clean, raw text output.     │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 3. Proposed Solution / Refactoring Strategy                           │
│                                                                            │
│   ### 3.1. High-Level Design / Architectural Overview                      │
│                                                                            │
│   The core change involves introducing an  OutputMode  concept to          │
│   runQuery  to control styling and decorative output.                      │
│                                                                            │
│   1. Determine Output Mode: A new boolean parameter ( rawOutput  or        │
│   similar) should be passed to  runQuery  based on whether the output      │
│   is redirected via  -o  or if the user is using  stdin  pipe and the      │
│   output is not to a TTY.                                                  │
│   2. Conditional Output: Inside  runQuery :                                │
│     * If  rawOutput  is true: suppress all spin/status messages (or        │
│     direct them to a verbose log/suppressed logger), and print only        │
│     the raw text to the final destination (file or  os.Stdout ).           │
│     * If  rawOutput  is false (standard TTY interaction): proceed with     │
│     full decorative rendering ( lipgloss ,  glamour ).                     │
│                                                                            │
│                                                                            │
│   ### 3.2. Key Components / Modules                                        │
│                                                                            │
│   The main modification will be in  internal/commands/  to handle the      │
│   flow, and a new flag may be introduced for explicit control.             │
│                                                                            │
│   *  internal/commands/root.go : Modify  runQuery  signature and           │
│   determine  rawOutput  state.                                             │
│   *  internal/commands/query.go :                                          │
│     * Update  runQuery  to accept and respect the  rawOutput  mode.        │
│     * Bypass decorative printing in  runQuery  when  rawOutput  is         │
│     true.                                                                  │
│   *  internal/commands/  (Utility): A new function to check if  stdout     │
│   is a terminal (TTY) is needed to correctly infer raw mode for            │
│   STDIN/STDOUT pipes.                                                      │
│                                                                            │
│   ### 3.3. Detailed Action Plan / Phases                                   │
│                                                                            │
│   #### Phase 1: Preparation & Raw Output Determination (S)                 │
│                                                                            │
│   Objective(s): Establish the logic for detecting raw output mode.         │
│                                                                            │
│    Task             | Rationale/Goal    | Esti… | Deliverable/Crit…        │
│   ------------------+-------------------+-------+-------------------       │
│    1.1: New TTY     | Need a reliable   | S     | New function             │
│    Check Utility    | way to check if   |       |  isStdoutTTY() bo        │
│                     |  os.Stdout  is a  |       | ol  in                   │
│                     | terminal/TTY,     |       |  internal/command        │
│                     | similar to        |       | s/  (or                  │
│                     |  getTerminalWidth |       |  internal/util/ )        │
│                     |  .                |       | using                    │
│                     |                   |       |  golang.org/x/ter        │
│                     |                   |       | m .                      │
│    1.2: Update      | Determine the     | S     | Update                   │
│     root.go  logic  | correct           |       |  rootCmd.RunE            │
│                     |  rawOutput  state |       | to:                      │
│                     | based on context  |       |  isTTY := isStdou        │
│                     | for  runQuery .   |       | tTTY() ;                 │
│                     |                   |       |  isPipeOutput :=         │
│                     |                   |       | hasStdin && (outp        │
│                     |                   |       | utFlag == "") &&         │
│                     |                   |       | !isTTY ;                 │
│                     |                   |       |  isFileOutput :=         │
│                     |                   |       | outputFlag != ""         │
│                     |                   |       | ;                        │
│                     |                   |       |  rawMode := isPip        │
│                     |                   |       | eOutput || isFile        │
│                     |                   |       | Output . Pass            │
│    1.3: Update      | Modify  runQuery  | S     | Change                   │
│     query.go        | to accept the new |       |  func runQuery(pr        │
│    signature        | mode parameter.   |       | ompt string) erro        │
│                     |                   |       | r  to                    │
│                     |                   |       |  func runQuery(pr        │
│                     |                   |       | ompt string, rawO        │
│                     |                   |       | utput bool) error        │
│                     |                   |       |  . Update calls          │
│                     |                   |       | in  root.go .            │
│                                                                            │
│   #### Phase 2: Implement Raw Output Suppression in  query.go  (M)         │
│                                                                            │
│   Objective(s): Modify  runQuery  to bypass decorative elements and        │
│   print only raw content when in  rawOutput  mode.                         │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    2.1: Conditional   | Status messages     | | In  runQuery , wrap        │
│    Spinner/Status     | (connect, upload,   | | calls to                   │
│    Suppression        | done, clipboard)    | |  newSpinner ,              │
│                       | must be suppressed  | |  spin.start() ,            │
│                       | in raw mode, as     | |  spin.stopWithSucce        │
│                       | they pollute        | | ss() ,                     │
│                       | redirected output.  | |  spin.stopWithError        │
│                       |                     | | () , and                   │
│                       |                     | | clipboard/warning          │
│                       |                     | | messages in a              │
│                       |                     | |  if !rawOutput { ..        │
│                       |                     | | . }  block.                │
│    2.2: Conditional   | Create separate     | | Inside  runQuery ,         │
│    Output Path        | branches for output | | after                      │
│    Implementation     | based on            | |  output, err := cli        │
│                       |  rawOutput  mode.   | | ent.GenerateContent        │
│                       |                     | | (...) :                    │
│                       |                     | |  if rawOutput { ...        │
│                       |                     | |  } else { ... } .          │
│    2.3: Raw Content   | If  rawOutput  is   | | If                         │
│    Output to          | true: skip all      | |  outputFlag == "" &        │
│    File/Stdout        | decoration and      | | & rawOutput  is            │
│                       | print only          | | true, write  text          │
│                       |  text := output.Tex | | to  os.Stdout  and         │
│                       | t()  to  os.Stdout  | | return. If                 │
│                       | (if not  -o ) or    | |  outputFlag != ""          │
│                       | proceed with file   | | (file output),             │
│                       | write (which is     | | ensure only the            │
│                       | already raw, but    | | success message (if        │
│                       | suppress the        | | any) is                    │
│                       |  successMsg  if it  | | conditionally              │
│                       | goes to             | | printed/suppressed.        │
│    2.4: Decorative    | If  rawOutput  is   | | Wrap the existing          │
│    Output Refactor    | false, ensure the   | |  getTerminalWidth()        │
│                       | current decorated   | |  ,                         │
│                       | logic (TTY only) is | |  assistantLabelStyl        │
│                       | executed as-is,     | | e ,                        │
│                       | printing to         | |  thoughtsStyle ,           │
│                       |  os.Stdout  and     | |  render.MarkdownWit        │
│                       |  os.Stderr  as      | | hWidth , and               │
│                       | before.             | |  assistantBubbleSty        │
│                       |                     | | le  logic in an            │
│                       |                     | |  else  block or            │
│                                                                            │
│   #### Phase 3: Verification & Cleanup (S)                                 │
│                                                                            │
│   Objective(s): Ensure all modes function as intended and code is          │
│   clean.                                                                   │
│                                                                            │
│    Task               | Rationale/Goal     | … | Deliverable/Crite…        │
│   --------------------+--------------------+---+--------------------       │
│    3.1: Verify  -o    | Check that         | S | Successful test           │
│    behavior           |  geminiweb "prompt |   | case: clean file          │
│                       | " -o file.md       |   | output, quiet             │
│                       | creates a raw file |   | console.                  │
│                       | and prints nothing |   |                           │
│                       | decorative to the  |   |                           │
│                       | console            |   |                           │
│                       | ( stdout / stderr  |   |                           │
│    3.2: Verify Pipe   | Check that `echo   | g | S                         │
│    behavior           | "prompt"           | e |                           │
│                       |                    | m |                           │
│                       |                    | i |                           │
│                       |                    | n |                           │
│                       |                    | i |                           │
│                       |                    | w |                           │
│                       |                    | e |                           │
│                       |                    | b |                           │
│                       |                    | ` |                           │
│                       |                    | ( |                           │
│                       |                    | i |                           │
│                       |                    | f |                           │
│                       |                    | n |                           │
│                       |                    | o |                           │
│                       |                    | t |                           │
│                       |                    | T |                           │
│                       |                    | T |                           │
│                       |                    | Y |                           │
│                       |                    | ) |                           │
│                       |                    | p |                           │
│                       |                    | r |                           │
│                       |                    | i |                           │
│                       |                    | n |                           │
│                       |                    | t |                           │
│                       |                    | s |                           │
│                       |                    | o |                           │
│                       |                    | n |                           │
│                       |                    | l |                           │
│                       |                    | y |                           │
│                       |                    | t |                           │
│                       |                    | h |                           │
│                       |                    | e |                           │
│                       |                    | r |                           │
│                       |                    | a |                           │
│                       |                    | w |                           │
│                       |                    | r |                           │
│                       |                    | e |                           │
│                       |                    | s |                           │
│                       |                    | p |                           │
│                       |                    | o |                           │
│                       |                    | n |                           │
│                       |                    | s |                           │
│                       |                    | e |                           │
│                       |                    | t |                           │
│                       |                    | e |                           │
│                       |                    | x |                           │
│                       |                    | t |                           │
│                       |                    | . |                           │
│    3.3: Verify TTY    | Check that         | S | Successful test           │
│    behavior           |  geminiweb "prompt |   | case: Decorated           │
│                       | "  (regular        |   | output, status            │
│                       | command) still     |   | messages visible.         │
│                       | prints the full    |   |                           │
│                       | decorated output   |   |                           │
│                       | with status        |   |                           │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 4. Key Considerations & Risk Mitigation                               │
│                                                                            │
│   ### 4.1. Technical Risks & Challenges                                    │
│                                                                            │
│   * Risk: Over-suppression of Status/Error Messages: Suppressing all       │
│   output (spinners, success, errors) on  stderr  in raw mode might         │
│   hide critical authentication failures or usage limit messages from       │
│   the user, especially when the main output is redirected.                 │
│   * Mitigation: Targeted Redirection: In raw mode, only suppress           │
│   decorative success/spinner messages. Ensure that critical errors (       │
│   spin.stopWithError()  and  fmt.Errorf ) are still written to  os.        │
│   Stderr  regardless of  rawOutput  status, as this is the standard        │
│   Unix convention for errors in pipe/redirect scenarios.                   │
│                                                                            │
│   ### 4.2. Dependencies                                                    │
│                                                                            │
│   * Internal:  internal/commands/root.go  and  internal/commands/query.    │
│   go  must be updated in tandem.                                           │
│   * External: Reliable function for TTY check (using  golang.              │
│   org/x/term  as already imported).                                        │
│                                                                            │
│   ### 4.3. Non-Functional Requirements (NFRs) Addressed                    │
│                                                                            │
│   * Usability: Improved Pipe/Redirection UX. Satisfies the user            │
│   requirement for clean, raw output when integrating with other            │
│   command-line tools (piping, file redirection).                           │
│   * Maintainability: Separation of Concerns. Isolating the rendering       │
│   logic based on  rawOutput  mode improves the separation between the      │
│   core logic ( client.GenerateContent ) and the presentation logic (       │
│   lipgloss ,  glamour ).                                                   │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 5. Success Metrics / Validation Criteria                              │
│                                                                            │
│   * The output to the file specified by  -o  contains only the raw         │
│   model response text.                                                     │
│   * When executing  echo "prompt" | geminiweb , the output contains        │
│   only the raw model response text (assuming  stdout  is not a TTY).       │
│   * When using the  -o  flag or piping to a non-TTY, no decorative         │
│   elements (labels, thoughts, bubbles, status spinners) are printed to     │
│   either  stdout  or  stderr  (excluding critical errors).                 │
│   * The default execution mode ( geminiweb "prompt" ) remains fully        │
│   decorated.                                                               │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 6. Assumptions Made                                                   │
│                                                                            │
│   * The absence of TTY ( !isStdoutTTY() ) is a sufficient heuristic        │
│   for deciding to use  rawOutput  when not explicitly saving to a file     │
│   ( -o ).                                                                  │
│   * In raw mode, the user expects the model response to be printed         │
│   directly to  os.Stdout , unless the  -o  flag redirects it to a file,    │
│   in which case nothing decorative should be printed to  os.Stdout /       │
│   os.Stderr .                                                              │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 7. Open Questions / Areas for Further Investigation                   │
│                                                                            │
│   * Should a dedicated flag like  --raw  or  --decorate=false  be          │
│   introduced to allow the user to force raw output even to a TTY, or       │
│   force decorated output even to a pipe? (Currently out of scope, but      │
│   good discussion point).                                                  │
│   * How to handle images ( -i  flag) in raw mode? The output text          │
│   already contains minimal image references; the plan assumes the raw      │
│   text is sufficient and no special image handling is required for raw     │
│   output.                                                                  │
╰────────────────────────────────────────────────────────────────────────────╯