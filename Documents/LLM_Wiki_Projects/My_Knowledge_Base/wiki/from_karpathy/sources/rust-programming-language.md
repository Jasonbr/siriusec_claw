# Rust (programming language)

## Summary

Wikipedia article on Rust, a general-purpose programming language emphasizing performance, type safety, concurrency, and memory safety. Covers the language's history from Graydon Hoare's personal project at Mozilla (2006) through the Rust Foundation era (2021–present), its syntax and features, and adoption across the software industry.

## Source Metadata

- Raw path: `raw/sources/20260514-092056-rust--programming-language-.md`
- Author: Wikipedia contributors
- Published: Live Wikipedia article (captured 2026-05-14)
- Ingested: 2026-05-14

## Key Points

- Rust is a multi-paradigm language (concurrent, functional, generic, imperative, structured) with C-like syntax and strong static typing.
- Memory safety is enforced at compile time via an ownership system and borrow checker, eliminating the need for a garbage collector.
- Created by Graydon Hoare at Mozilla in 2006; first stable release (1.0) shipped May 2015.
- Following Mozilla layoffs in 2020, the Rust Foundation was formed (2021) with sponsors including AWS, Google, Huawei, Microsoft, and Mozilla.
- Influenced by C++, OCaml, Haskell, Erlang, and others; targets frustrated C++ developers seeking safety without sacrificing performance.
- Widely adopted in web services, system software, browsers (Firefox, Servo), and by major tech companies.

## Evidence / Notes

- The borrow checker tracks object lifetimes of references at compile time to prevent dangling pointers and data races.
- Variables are immutable by default; mutability requires the `mut` keyword.
- Rust uses affine types: each value may be used at most once, enforcing software fault isolation.
- The standard library is divided into `core`, `alloc`, and `std`, with `#![no_std]` support for embedded devices.
- Polymorphism is achieved through traits, generics, and trait objects (static vs dynamic dispatch).
- Error handling uses `Option<T>` and `Result<T, E>` types rather than exceptions or null pointers.

## Related Entities

- [rust-programming-language](../entities/rust-programming-language.md)
- [mozilla](../entities/mozilla.md)
- [graydon-hoare](../entities/graydon-hoare.md)
- [rust-foundation](../entities/rust-foundation.md)

## Related Concepts

- [memory-safety](../concepts/memory-safety.md)
- [ownership-system](../concepts/ownership-system.md)
- [borrow-checker](../concepts/borrow-checker.md)
- [type-safety](../concepts/type-safety.md)

## Wiki Updates

- Created [rust-programming-language](../entities/rust-programming-language.md)
- Created [mozilla](../entities/mozilla.md)
- Created [graydon-hoare](../entities/graydon-hoare.md)
- Created [rust-foundation](../entities/rust-foundation.md)
- Created [memory-safety](../concepts/memory-safety.md)
- Created [ownership-system](../concepts/ownership-system.md)
- Created [borrow-checker](../concepts/borrow-checker.md)
- Created [type-safety](../concepts/type-safety.md)
- Updated [overview](../overview.md)
- Updated [index](../index.md)

## Open Questions

- How will the Rust Foundation's trademark policy evolution affect community trust and corporate adoption?
- Can Rust's compile-time guarantees be extended to formally verify security properties beyond memory safety?
- What is the long-term trajectory of Rust in domains currently dominated by C (e.g., Linux kernel, embedded RTOS)?
