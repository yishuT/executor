This is a golang thread pool implementation.

Properties:

1. Fix number of workers
2. FIFO, jobs are executed in the order of submittion
3. If number of jobs in the queue is equal to max queueu capacity, new submittion will be rejected.
4. Jobs sitting in the queue longer than max queue timeout are dropped. (circuit breaking)
5. Executor shutdown gracefully until shutdown timeout

TODO:
1. benchmark
2. running job cancelation
3. job scheduled by priority instead of submittion time
4. dynamic number of workers based on work load
