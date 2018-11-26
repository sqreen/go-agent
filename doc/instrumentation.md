# Run Time Instrumentation Techniques

Go is a statically typed and compiled language, which, by design, doesn't have
anything like what dynamic languages may provide to perform code transformation
at run time. Compilers make the most out of statically typed languages to
generate the best binary program for the targeted hardware architecture. For
example, they avoid dynamic calls through indirection as long as it is possible
when actual types are know at compilation time. Moreover, Go uses a specific ABI
to handle its special language features such as goroutines or multiple return
values.

This document analyses and compare the different approaches to dynamically
instrument compiled programs, with special attention to Sqreen's agent
coarsed-grained requirements: performance, reliabily and robustness.

## Wording

The wording used here matches the one used in the bibliography which mainly
comes from the performance monitoring field.

- Tracepoint:  
  A point in the instrumented software that can be "traced". They can thus be
  either statically defined or dynamically injected. Its goal is to get some
  context and call one or several probe functions.
  
- Probe:  
  A piece of code that can be attached to a tracepoint and executed when the
  instrumented software executes the tracepoint.

Therefore, instrumentation involves both having tracepoints and probe functions.

## Analysis

### Introduction

Tracepoints and probe functions can be either dynamically or statically
injected. The resulting combinations give different approaches with their own
pros & cons. This analysis goes through all of them to better understand what
are Sqreen's options to implement the Go Agent, and possibly any compiled binary
program.

We distinguish the *user* who wants to instrument its binary software from the
*instrumenter* who provides the instrumentation solution. The *user* is looking
for the least intrusive solution, with the least overhead performances, and the
easiest *user* experience, ie. idealy not having to modify its source code.

On the *instrumenter*'s point of view, the three main approaches are:

- The least effort which involves avoiding code transformation, which involves
  code relocation, by using dynamic instrumentation, ie. software breakpoints!
  Therefore, this technique requires kernel handlers.
  
- The fully user-space approach which involves run time code transformation and
  relocation. But the reverse engineering approach of it makes its usage in
  production environment pretty dangereous (ambiguous variable-length
  instructions, etc.).

- The stability and performance approach which involves user-defined static
  tracepoints (defined in the source-code), along with dynamic probe functions.

Note the tradeoff here: code intrusivness vs run time intrusivness. The analysis
will reflect it.

### Analysis Table

The mainly adopted techniques have been analysed in the following sheet:

[Google Sheet][eval]

### Conclusion

FIXME: after monday's reunion. we could do a matrix: for each technique, sum-up
to what extent do they answer to our needs (robustness, performance, reliabily)?

## Evaluation

So obviously, nothing precisely solves every Sqreen's requirement, but we
definitely can mix the best options together to obtain the best implementation.

# Bibliography

- [A thorough introduction to eBPF][ebpf-intro]
- [Using user-space tracepoints with BPF][lwn-usdt]
- [DynInst - Anywhere, Any-Time Binary Instrumentation][dyninst-abstract]
- [DynInst - An API for Runtime Code Patching][dyninst-api]
- [Toward the Deconstruction of Dyninst][dyninst-deconstruction]
- [Uprobe-tracer: Uprobe-based Event Tracing][uprobe]

[dyninst-abstract]: http://ftp.cs.wisc.edu/paradyn/papers/Bernat11AWAT.pdf
[ebpf-intro]: https://lwn.net/Articles/740157/
[lwn-usdt]: https://lwn.net/Articles/753601/
[eval]: https://docs.google.com/spreadsheets/d/1J59JA7LlZ0y_MP1rmTb0q2h24cauKzHvX1gkOubyy9U
[dyninst-api]: http://www.cs.umd.edu/~hollings/papers/apijournal.pdf
[dyninst-deconstruction]: http://ftp.cs.wisc.edu/paradyn/papers/Ravipati07SymtabAPI.pdf
[uprobe]: https://www.kernel.org/doc/Documentation/trace/uprobetracer.txt
