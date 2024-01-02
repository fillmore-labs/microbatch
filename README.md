# Micro Batcher

[![Go Reference](https://pkg.go.dev/badge/fillmore-labs.com/microbatch.svg)](https://pkg.go.dev/fillmore-labs.com/microbatch)

Micro-batching is a technique often used in stream processing to achieve near real-time computation
while reducing the overhead compared to single record processing. It balances latency versus throughput
and enables simplified parallelization while optimizing resource utilization.

See also the definition in the [Hazelcast Glossary](https://hazelcast.com/glossary/micro-batch-processing/) and
explanation by [Jakob Jenkov](https://jenkov.com/tutorials/java-performance/micro-batching.html).
Popular examples are [Spark Structured Streaming](https://spark.apache.org/docs/latest/structured-streaming-programming-guide.html#overview), [Apache Kafka](https://kafka.apache.org/documentation/#upgrade_11_message_format) and others.

## Usage

### Implement `Job` and `JobResult`

```go
type (
	Job struct {
		ID      string
		Request string
	}

	JobResult struct {
		ID       string
		Response string
	}
)

func correlateRequest(j *Job) string      { return j.ID }
func correlateResult(r *JobResult) string { return r.ID }
```

### Implement the Batch Processor

```go
type RemoteProcessor struct{}

func (*RemoteProcessor) ProcessJobs(jobs []*Job) ([]*JobResult, error) {
    ... // Send the jobs downstream for processing and return the results
}
```

### Use the Batcher

```go
// Initialize
processor := &RemoteProcessor{}
opts := []microbatch.Option{microbatch.WithSize(3), microbatch.WithTimeout(10 * time.Millisecond)}
batcher := microbatch.NewBatcher(processor, correlateRequest, correlateResult, opts...)

var wg sync.WaitGroup

// Process jobs
ctx := context.Background()
wg.Add(1)
go func() {
    defer wg.Done()
    if result, err := batcher.ExecuteJob(ctx, &Job{ID: "1"}); err == nil {
        fmt.Println(result)
    }
}()

// Shut down
wg.Wait()
batcher.Shutdown()
```

## Links
- [Example project calling AWS Lambda](https://github.com/fillmore-labs/microbatch-lambda)
