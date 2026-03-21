package deno

import _ "embed"

//go:embed ts/runner.ts
var runnerScript []byte

//go:embed ts/worker_wrapper.ts
var workerWrapperScript []byte
