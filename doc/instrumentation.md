# Go Agent's Dynamic Instrumentation

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Table of Contents**

- [Go Agent's Dynamic Instrumentation](#go-agents-dynamic-instrumentation)
    - [Run Time Instrumentation Techniques](#run-time-instrumentation-techniques)
        - [Wording](#wording)
        - [Analysis](#analysis)
        - [Evaluation](#evaluation)
            - [Fully user-space dynamic binary instrumentation](#fully-user-space-dynamic-binary-instrumentation)
            - [Fully user-space static binary instrumentation](#fully-user-space-static-binary-instrumentation)
            - [Trap-based dynamic binary instrumentation](#trap-based-dynamic-binary-instrumentation)
            - [Conclusion](#conclusion)
    - [Binary instrumentation for Sqreen's Go Agent](#binary-instrumentation-for-sqreens-go-agent)
        - [Evaluation](#evaluation-1)
        - [Binary instrumentation of user-compiled files](#binary-instrumentation-of-user-compiled-files)
            - [Option 1](#option-1)
            - [Option 2](#option-2)
            - [Option 3](#option-3)
        - [Binary instrumentation of external dynamic libraries](#binary-instrumentation-of-external-dynamic-libraries)
            - [Option 1](#option-1-1)
            - [Option 2](#option-2-1)
    - [Bibliography](#bibliography)

<!-- markdown-toc end -->

## Run Time Instrumentation Techniques

Go is a statically typed and compiled language, which *by design* doesn't have
anything like what dynamic languages may provide to perform code transformation
at run time. Compilers make the most out of statically typed languages to
generate the best binary program for the targeted hardware architecture. For
example, compilers avoid dynamic calls through indirection tables (tables of
function pointers) as long as actual types are know at compilation time.
Moreover, Go uses a specific ABI to handle its special language features such as
goroutines or multiple return values.

This document analyses and compares the different approaches to dynamically
instrument compiled programs, with special attention to Sqreen's agent
coarse-grained requirements: performance, reliability and portability.

### Wording

The wording used here matches the same as in the bibliography which mainly comes
from the performance analysis field:

- Tracepoint:  
  A point in the instrumented software that can be *traced*. They can thus be
  either statically defined or dynamically injected. Its goal is call one or
  several probe functions.
  
- Probe:  
  A piece of code that can be attached to a tracepoint and executed when the
  instrumented software executes the tracepoint.

Also, *static binary instrumentation* refers to instrumentation performed on the
binary executable file, from compilation to linkage, but also final executable
file rewriting. *Dynamic binary instrumentation* refers to instrumentation
performed at run time.

### Analysis

Dynamic instrumentation involves having tracepoints and probe functions. The
following analysis of existing techniques will show they can be either
dynamically or statically injected into running programs, with their own pros &
cons. This analysis goes through all of them to better understand what are
Sqreen's possible options to implement the Go Agent, and possibly any compiled
binary program.

We distinguish here the *user* who wants to instrument its binary software from
the *instrumenter* who provides the instrumentation solution. The *user* is
looking for the least intrusive solution, with the least overhead performances,
and the easiest *user* experience, ie. ideally not having to modify its source
code.

The mainly adopted techniques have been qualitatively analyzed in the linked
[Google Sheet][eval].

### Evaluation

The three main techniques to call a probe function use either traps, trampolines
or JIT recompilation. On the *instrumenter*'s point of view the *user*'s
overall experience is totally different depending on which technique is used.
This conclusion summarizes what the *user* experience is depending on the chosen
techniques.

#### Fully user-space dynamic binary instrumentation

The fully user-space approach involves either:

- code transformation and relocation at run time:  
  Given a code location to instrument, (i) copy and relocate the instruction or
  block of interdependent instructions to another place, (ii) create a
  trampoline jumping to this relocated block (trampolines allow to write jumps
  to absolute addresses, while 1-instruction jump are relative and thus limited
  by the size of the instruction), (iii) write to the instrumentation point a
  branch instruction to the trampoline. Examples of such tools are DynInst and
  DynamoRIO. The disassembly this technique performs at run time makes its usage
  in production environments very dangerous and very likely to fail (eg.
  ambiguous parsing of variable-length instructions, overlapping function,
  etc.), along with its dependency to the ISA and ABI. A simple look at the
  [endless list of limitations][dynamorio-limitations], despite the huge number
  of years of work, shows how hard it can be to correctly implement.

- JIT recompilation:  
  Disassemble the running binary program at run time to create an abstract
  intermediate representation (IR) of it and be able to virtually modify it to
  insert tracepoints. It can then be dynamically recompiled into actual machine
  assembly code, cached, and simply executed. This technique can be seen as
  converting the binary program into bytecode, and having a VM to run it.
  Examples of tools using this technique are Valgrind, Qemu or Pin. This
  technique can be optimized by only recompiling instrumented parts and not the
  entire running program. Note that a cached implementation involves having
  trampolines to the recompiled code.
  
We consider the more robust technique here is JIT recompilation by assuming the
disassembly into an IR is less complex than the disassembly for code relocation

> TODO: to be further specified with more insights on the two implementations

Note that executable file rewriting, even though statically performed, uses
these same techniques. The difference being that the disassembling and analysis
is performed once and the resulting file doesn't include this initialization
overhead.

#### Fully user-space static binary instrumentation

To get rid of previously listed drawbacks, mainly coming from the complex
disassembling steps, the opposite approach is to offer a stable and efficient
*user* experience by statically injecting tracepoints into the program, before
it executes. This technique involves either:

- Source-code modification by the user or automated by the compilation toolchain
  to insert tracepoints.

- Link-time injection replacing calls to dynamic library functions.

The underlying implementation of the injected tracepoint can then vary, either:

- Direct calls to probe functions, that may be guarded by semaphores (eg. USDT)
  to enable and disable them.

- Insert a single no-op instruction in place of the tracepoint, along with
  metadata describing their locations. So this technique needs to be combined
  with another probing technique described in this document. It can be seen as a
  way to add some "blank bytes" in the binary code, and thus get rid of run time
  code relocation because this no-op instruction can simply be overwritten
  without any concern. So it's a way to just jump straight to a probe function
  through a trampoline.

#### Trap-based dynamic binary instrumentation

The straightforward and least effort technique is to simply use software traps.
On the *instrumenter* side, they offer standardized and abstracted by the
operating-system to trap the execution at run time, while on the *user* side,
they offer the smallest binary transformation impact. A software trap is indeed
nowadays synonym of software breakpoint (legacy or pure RISC architectures may
not provide software breakpoints, in which case it is implemented using any
other software fault). For example, on Intel architectures, they are implemented
with a software breakpoint, which is a special single-byte opcode that can be
inserted into any Intel instruction.

The major drawback is of course the fact that software traps are handled by a
kernel interrupt handler, thus with a much higher overhead than user-space
techniques, no matter how much optimized. Other overheads also come from the
fact it takes the CPU interrupt path, which is very hardware-dependent no matter
how much software-optimized. Note that we consider user-space interrupt
handlers, directly called by the CPU on interrupt, not likely to happen in
modern general-purpose operating-systems used by our clients.

Also note that the reason why uprobes are efficient despite using this
technique is because the probe function is executed in kernel-space by the eBPF
VM. So the entry into the kernel is not useless, and there are not
user-kernel-user round-trips.

#### Conclusion

These existing techniques are a trade-off between:

1. The implied overhead (mainly time overhead, as memory overhead is
   only not negligible with JIT recompilation).
1. What can be hooked.
1. Robustness and reliability of the technique.

## Binary instrumentation for Sqreen's Go Agent

This section fist evaluates the analyzed techniques, and finally goes only
through techniques compliant to our requirements.

### Evaluation

Nothing perfectly fits our Sqreen's requirements, but we can definitely get
the most out of them to offer our *users* the best performance, reliability and
portability experiences.

|                | Performance | Reliability | Portability |
|----------------|-------------|-------------|-------------|
| Dynamic instr. | 🗸          |             |             |
| Static instr.  | 🗸          | 🗸          | 🗸          |
| Trap instr.    |             | 🗸          | 🗸          |

Notes:

- **Performance** is about having the minimum overhead, both when a hookpoint is
  enabled or disabled. Trap-based techniques are excluded because too slow when
  enabled because of the kernel round-trips.

- **Reliability** is about having a stable run time technique, unlikely to fail
  or crash the instrumented program. Hence the exclusion of dynamic
  instrumentation which involves disassembling parts of the program at run time,
  too harmful and likely incomplete involving fatal non-recoverable run time
  errors.
  
- **Portability** is about having a portable technique, stable among *user*'s
  environments. Dynamic instrumentation is excluded because too
  hardware-dependent. The trap-based technique is compliant because abstracted
  by the operating-system, and they all provide trap APIs.

### Binary instrumentation of user-compiled files

The following list of choices can apply to static calls, dynamic calls, calls to
libraries that were part of the *user* compilation. External dynamic libraries
are out of this scope and managed separately.

#### Option 1

Provide to the *user* an Agent API to manually insert in its source-code
hookpoints into strategic locations.

#### Option 2

Compilation-time instrumentation inserting hookpoints on entry and exit of
functions and methods. The list of functions to be hooked could be listed in a
sqreen-defined configuration file, or every single compiled function.

These entry/exit hookpoints can be designed either as:

1. A call to an generic sqreen hookpoint handler function guarded by a semaphore
   (one per hookpoint). The major benefit of this solution is that it is
   entirely user-space and purely implemented in Go, therefore with a nice level
   of hardware abstraction (no need for binary instruction patching).

1. A no-op instruction along with its location metadata: to be patched at run
   time. The benefit here is the minimized performance overhead, while the
   drawback is the need for privileges to be able to write into the process code
   memory region.
   
> TODO: We need to benchmark how close to the no-op is the semaphore guard +
> static branch prediction (asm hints to tell the branch prediction what path to
> take when it doesn't know) technique. It could remove the no-op option if it
> performs similarly!

#### Option 3

Smart JIT recompilation of instrumented blocks, exactly like QEMU does when
replacing privileged instructions, when a function needs to be instrumented:

1. Disassemble its impacted blocks into an intermediate representation (IR).
   This IR is an abstracted ISA, which is equivalent to bytecode language, so we
   could use an existing IR/bytecode.

1. Modify this IR to instrument it. Note that this is not longer limited to
   entry/exit instrumentation.

1. Patch the function to trampoline into the JIT IR recompilation. The resulting
   assembly must be cached so that the JIT overhead happen only once, during the
   first call.

Warning: when performed onto dynamic libraries, it should only impact the *user*
program and not other running processes also using them.

### Binary instrumentation of external dynamic libraries

External dynamic libraries are part of the *user* program's environment, and not
part on the *user* compilation. Therefore, they cannot be instrumented by
previously listed static binary instrumentation techniques. Note the need to be
able to catch nested calls to dynamic libraries (eg. libmysql.so uses libc.so).

#### Option 1

Replace the linkage table entries we want to instrument with the address of an
agent function.

> TODO:
> - check what can be performed at run time without privileges. Eg. is it
>   read-only?
> - check if we can link against the PLT address, or if we instead need to read
>   the executable file.
> - check we can do this in Go's init stage.

#### Option 2

The previously detailed JIT option should also apply to dynamic libraries, with
the downside of being shared with other processes.

## Bibliography

- [A thorough introduction to eBPF][ebpf-intro]
- [Using user-space tracepoints with BPF][lwn-usdt]
- [DynInst - Anywhere, Any-Time Binary Instrumentation][dyninst-abstract]
- [DynInst - An API for Runtime Code Patching][dyninst-api]
- [Toward the Deconstruction of Dyninst][dyninst-deconstruction]
- [Uprobe-tracer: Uprobe-based Event Tracing][uprobe]
- [Dynamic Binary Analysis and Instrumentation or Building Tools is Easy][thesis-valgrind]

[dyninst-abstract]: http://ftp.cs.wisc.edu/paradyn/papers/Bernat11AWAT.pdf
[ebpf-intro]: https://lwn.net/Articles/740157/
[lwn-usdt]: https://lwn.net/Articles/753601/
[eval]: https://docs.google.com/spreadsheets/d/1J59JA7LlZ0y_MP1rmTb0q2h24cauKzHvX1gkOubyy9U
[dyninst-api]: http://www.cs.umd.edu/~hollings/papers/apijournal.pdf
[dyninst-deconstruction]: http://ftp.cs.wisc.edu/paradyn/papers/Ravipati07SymtabAPI.pdf
[uprobe]: https://www.kernel.org/doc/Documentation/trace/uprobetracer.txt
[dynamorio-limitations]: http://dynamorio.org/docs/release_notes.html#sec_limits
[thesis-valgrind]: http://www.valgrind.org/docs/phd2004.pdf
