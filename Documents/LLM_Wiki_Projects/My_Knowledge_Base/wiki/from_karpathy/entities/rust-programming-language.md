# Rust (programming language)

## Summary

Rust is a general-purpose, multi-paradigm programming language designed for performance, reliability, and productivity. It enforces memory safety and thread safety at compile time without requiring a garbage collector, making it suitable for systems programming, web services, and embedded development.

## Key Points

- First stable release (1.0) in May 2015; latest stable releases follow a 6-week train model.
- Originally developed at Mozilla; governance transferred to the independent Rust Foundation in 2021.
- Syntax influenced by C and C++; type system influenced by OCaml and Haskell.
- Core innovation is the ownership system combined with the borrow checker for compile-time memory safety.
- Widely adopted by Amazon, Google, Microsoft, Meta, and others for performance-critical infrastructure.
- Used in the Firefox browser (Servo/Quantum projects), AWS services, and the Linux kernel.

## Evidence / Notes

- Source: [rust-programming-language Wikipedia article](../sources/rust-programming-language.md)
- Mozilla sponsored the project starting in 2009; creator Graydon Hoare began work in 2006.
- The compiler was originally written in OCaml (2006–2011) before becoming self-hosting.
- RFC process established in 2014 for community-driven language evolution.

## Links

- [mozilla](../entities/mozilla.md)
- [graydon-hoare](../entities/graydon-hoare.md)
- [rust-foundation](../entities/rust-foundation.md)
- [memory-safety](../concepts/memory-safety.md)
- [ownership-system](../concepts/ownership-system.md)
- [borrow-checker](../concepts/borrow-checker.md)
- [type-safety](../concepts/type-safety.md)

## Open Questions

- Will Rust achieve mainstream adoption in game development and real-time systems?
- How will the language evolve async/await and effect systems without compromising its core guarantees?
