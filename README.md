# Micro Batcher

[![Go Reference](https://pkg.go.dev/badge/fillmore-labs.com/microbatch.svg)](https://pkg.go.dev/fillmore-labs.com/microbatch)

Micro-batching is a technique often used in stream processing to achieve near real-time computation
while reducing the overhead compared to single record processing. It balances latency versus throughput
and enables simplified parallelization while optimizing resource utilization.

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
const batchSize = 5
const batchDuration = 1 * time.Millisecond
batcher := microbatch.NewBatcher(processor, correlateRequest, correlateResult, batchSize, batchDuration)

var wg sync.WaitGroup

// Submit jobs
wg.Add(1)
go func() {
	result, _ := batcher.ExecuteJob(ctx, &Job{ID: 1})
	wg.Done()
}()

// Shut down
wg.Wait()
batcher.Shutdown()
```


## Links
- [Example project calling AWS Lambda](https://github.com/fillmore-labs/microbatch-lambda)
