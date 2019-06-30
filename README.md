This is a golang thread pool implementation.

Usage:
```
executor, err := NewExecutor(DefaultExecutorParams())
executor.Submit(func() { /*do something*/ })
executor.Stop()
```

Properties:

1. Fix number of workers
2. FIFO, jobs are executed in the order of submittion
3. If number of jobs in the queue is equal to max queue capacity, new submittion will be rejected.
4. Jobs sitting in the queue longer than max queue timeout are dropped. (circuit breaking)
5. Executor shutdown gracefully until shutdown timeout

TODO:
1. benchmark
2. runnable interface
3. return values of runnable
4. running job cancelation
5. job scheduled by priority instead of submittion time
6. dynamic number of workers based on work load
7. Queuing system
8. Pre-execute hook
9. Stats store
