# Quark
## A Java-like language
Quark is a specific-purpose programming language that takes a lot of stuff from Java, like
- WORA
- Neat packaging (like `.jar`)
- Portability
- Bytecode

But, is completley written from scratch. No JVM, no nothing.
# Gluon/Glue
Gluon/Glue is the package system of Quark. It bundles source files and metadata into a glorified ZIP file. One can be created with
```shell
$ quark glue
```
And you can run one with
```shell
$ quark superglue [package].gluon
```
As well as being standalone executables (like `.jar`s), they can also be used as libraries/packages in other programs.

When a project is created, it makes a `pkgs` directory. Gluons can be dropped into this directory and will be treated as part of the source code.
# Bytecode and compilation
During compilation, the compiler will do the following
- Create `finished` variable
- Append the bytecode of existing Gluons in `pkgs` to `finished`
- Compile all the source to bytecode and append it to `finished`
- Convert `finished` into a Gluon
