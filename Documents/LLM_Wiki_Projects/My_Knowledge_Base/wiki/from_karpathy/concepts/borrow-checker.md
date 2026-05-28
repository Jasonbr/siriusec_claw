# Borrow Checker

## Summary

The borrow checker is Rust's compile-time analysis pass that enforces the ownership and borrowing rules. It ensures references are always valid and that mutable and immutable borrows do not overlap in ways that could cause data races or use-after-free bugs.

## Key Points

- The borrow checker tracks object lifetimes to ensure references never outlive the data they point to.
- It rejects programs where a reference could become invalid, such as returning a reference to a local variable.
- Lifetime parameters (e.g., `'a`) allow explicit annotation of reference validity when inference is insufficient.
- The checker prevents aliasing XOR mutation: either many immutable references exist, or one mutable reference exists, but not both at the same time.
- Errors from the borrow checker are a common learning hurdle for new Rust programmers.

## Evidence / Notes

- Source: [rust-programming-language Wikipedia article](../sources/rust-programming-language.md)
- Memory safety errors and data races are prevented by tracking object lifetimes at compile time.
- Lifetimes are inferred from source code locations (function, line, column) where a variable is valid.

## Links

- [rust-programming-language](../entities/rust-programming-language.md)
- [memory-safety](../concepts/memory-safety.md)
- [ownership-system](../concepts/ownership-system.md)

## Open Questions

- Are there plans to make borrow checker error messages more actionable for beginners?
- Can borrow checking be extended to higher-level abstractions like async/await state machines without false positives?
