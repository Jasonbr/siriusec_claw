# Ownership System

## Summary

The ownership system is Rust's core mechanism for managing memory and resources without a garbage collector. It consists of a set of rules enforced at compile time that govern how values are created, moved, borrowed, and destroyed.

## Key Points

- Every value in Rust has a single owner at any given time.
- Values can be moved between owners through assignment or function calls; the original owner can no longer use the value.
- Values can be borrowed immutably (`&T`) by multiple readers or mutably (`&mut T`) by exactly one writer, but never both simultaneously.
- When an owner goes out of scope, the value's destructor (`Drop`) runs automatically, releasing resources.
- This system enforces affine types: each value may be used at most once before being moved or dropped.

## Evidence / Notes

- Source: [rust-programming-language Wikipedia article](../sources/rust-programming-language.md)
- The ownership system was in place by 2010 and replaced the garbage collector by 2013.
- Ownership rules enable software fault isolation by making the owner solely responsible for correctness and deallocation.

## Links

- [rust-programming-language](../entities/rust-programming-language.md)
- [memory-safety](../concepts/memory-safety.md)
- [borrow-checker](../concepts/borrow-checker.md)

## Open Questions

- How do ownership semantics interact with circular data structures and self-referential types?
- Can ownership-based resource management be generalized to distributed or persistent systems?
