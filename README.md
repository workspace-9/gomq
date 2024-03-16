# gomq
[![Go Reference](https://pkg.go.dev/badge/github.com/workspace-9/gomq.svg)](https://pkg.go.dev/github.com/workspace-9/gomq)

Sockets with super-powers

## Motivation

ZeroMQ is a powerful communications library, making robust networking simple (retry logic, socket patterns, multiple bind/connect per socket, etc.)
Go is a powerful programming language, and having go libraries written is pure Go makes it super easy to cross compile for other platforms and operating systems.
Go also provides a powerful asynchronous networking library which is built seamlessly into the language (all in the `net` package.)
This would make a ZMQ implementation in pure Go a powerful library.

## Why build a new one when others are already out there?
1. It's fun

## What sets this implementation apart?
1. Highly modularized. Implementing new socket types, transports, mechanisms, etc. is super simple. In fact, they need not be implemented in this repository as the code follows a dependency injection structure.
2. Support for CurveZMQ which is not found in any other Go implementation (to my searching.)
3. Tiny. This implementation does not aim to follow the same code structure as the core C++ ZMQ implementation but rather attempts to achieve a similar feature set.
