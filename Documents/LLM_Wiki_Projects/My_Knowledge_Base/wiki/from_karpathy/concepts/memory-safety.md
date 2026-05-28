# Memory Safety

## Summary

Memory safety is a property of programming languages and systems that prevents bugs and vulnerabilities arising from invalid memory access, such as use-after-free, buffer overflows, and dangling pointers. Rust achieves memory safety at compile time without a garbage collector.

## Key Points

- Rust prevents all dangling pointers and data races through compile-time checks rather than runtime overhead.
- The ownership system ensures each value has exactly one owner; when the owner goes out of scope, the value is dropped.
- References (`&T` and `&mut T`) are guaranteed non-null and valid for their entire lifetime.
- Raw pointers (`*const T`, `*mut T`) opt out of safety guarantees and can only be dereferenced within `unsafe` blocks.
- Traditional systems languages (C, C++) require manual discipline or tools (ASan, Valgrind) to catch memory errors at runtime.

## Evidence / Notes

- Source: [rust-programming-language Wikipedia article](../sources/rust-programming-language.md)
- Rust's borrow checker tracks object lifetimes to enforce that references never outlive the data they point to.
- No implicit type conversion between most primitive types reduces accidental memory reinterpretation bugs.

## Links

- [rust-programming-language](../entities/rust-programming-language.md)
- [ownership-system](../concepts/ownership-system.md)
- [borrow-checker](../concepts/borrow-checker.md)

## Open Questions

- Can Rust's memory safety model be extended to prevent all concurrency bugs, or are some data races still expressible in `unsafe` code?
- How does Rust's approach compare to newer languages like Vale or Hylo that also pursue memory safety?
