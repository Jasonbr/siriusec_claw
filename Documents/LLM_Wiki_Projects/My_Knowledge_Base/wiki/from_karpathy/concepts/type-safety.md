# Type Safety

## Summary

Type safety is the property of a programming language that prevents type errors, such as treating an integer as a pointer or accessing undefined memory through an invalid type cast. Rust's strong, static type system with type inference provides type safety while maintaining ergonomic development.

## Key Points

- Rust is strongly and statically typed: all variable types must be known at compile time.
- Type inference (`let x = 5`) allows omitting explicit type annotations when the compiler can deduce the type.
- No implicit type conversion (coercion) between most primitive types; explicit casting uses the `as` keyword.
- The `Option<T>` and `Result<T, E>` types replace null pointers and exceptions, enforcing explicit handling of missing or erroneous values.
- Algebraic data types (`enum`, `struct`) and pattern matching (`match`, `if let`) enable exhaustive, type-safe handling of all possible states.

## Evidence / Notes

- Source: [rust-programming-language Wikipedia article](../sources/rust-programming-language.md)
- Rust's typing discipline includes affine, inferred, nominal, static, and strong typing.
- Assigning a value of one type to a differently typed variable causes a compilation error.

## Links

- [rust-programming-language](../entities/rust-programming-language.md)
- [memory-safety](../concepts/memory-safety.md)

## Open Questions

- How does Rust's type system compare to dependent type systems (e.g., Idris, Lean) in terms of expressiveness vs. complexity?
- Could Rust benefit from gradual typing or row polymorphism for increased flexibility?
