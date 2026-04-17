// Package cc owns the host C toolchain — detection, invocation, and
// the translation between MIR codegen output and a native binary.
//
// At W05 the package provides two functions:
//
//   - Detect() picks a C compiler by probing `$CC`, then the usual
//     suspects (cc, gcc, clang, cl) on the system PATH.
//   - Compile(cInput, outBinary) invokes the detected compiler in
//     C11 mode and produces a platform-native binary at outBinary.
//
// The detection policy is deterministic: within a single run,
// Detect returns the first compiler found in a fixed probe order.
// Tests that control the probe order (e.g. to simulate a host with
// only `cl.exe`) use the exported probe helpers in compiler_test.go.
package cc
