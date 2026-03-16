/*
 * Copyright © 2022 photowey (photowey@gmail.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package poolgo

// Package poolgo provides a lightweight goroutine pool with graceful shutdown,
// context-aware submission, typed async helpers, and recoverable task execution semantics.
//
// The package is designed for internal service code that needs a small,
// predictable worker pool rather than a general-purpose scheduling framework.
//
// Recommended usage:
//
//   - create one pool per component or service subsystem
//   - prefer SubmitTyped for tasks that return values
//   - call Shutdown during service teardown
//   - pass bounded contexts when submitting work to avoid indefinite queue waits
//
// Lifecycle semantics:
//
//   - running: accepts new work
//   - shutting_down: rejects new work and drains accepted tasks
//   - stopped: all workers have exited
//
// Panic semantics:
//
//   - task panics are recovered
//   - recovered panics are converted into errors via the configured PanicHandler
//   - worker capacity is preserved after a panic
