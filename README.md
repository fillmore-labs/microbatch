# Micro Batcher

[![Go Reference](https://pkg.go.dev/badge/fillmore-labs.com/microbatch.svg)](https://pkg.go.dev/fillmore-labs.com/microbatch)
[![Build Status](https://badge.buildkite.com/1d68e28b14ecbbd4e4066e61c25f81ef08a8237615f5d03a6a.svg)](https://buildkite.com/fillmore-labs/microbatch)
[![Test Coverage](https://codecov.io/gh/fillmore-labs/microbatch/graph/badge.svg?token=Sh0xNVeFCd)](https://codecov.io/gh/fillmore-labs/microbatch)
[![Maintainability](https://api.codeclimate.com/v1/badges/2ba503a6a37cfc77951c/maintainability)](https://codeclimate.com/github/fillmore-labs/microbatch/maintainability)
[![Go Report Card](https://goreportcard.com/badge/fillmore-labs.com/microbatch)](https://goreportcard.com/report/fillmore-labs.com/microbatch)
[![License](https://img.shields.io/github/license/fillmore-labs/microbatch)](https://github.com/fillmore-labs/microbatch/blob/main/LICENSE)

Micro-batching is a technique often used in stream processing to achieve near real-time computation
while reducing the overhead compared to single record processing. It balances latency versus throughput
and enables simplified parallelization while optimizing resource utilization.

See also the definition in the [Hazelcast Glossary](https://hazelcast.com/glossary/micro-batch-processing/) and
explanation by [Jakob Jenkov](https://jenkov.com/tutorials/java-performance/micro-batching.html).
Popular examples
are [Spark Structured Streaming](https://spark.apache.org/docs/latest/structured-streaming-programming-guide.html#overview), [Apache Kafka](https://kafka.apache.org/documentation/#upgrade_11_message_format)
and others.

## Usage

### Implement `Job` and `JobResult`

```go
type (
	Job struct {
		ID      string `json:"id"`
		Request string `json:"body"`
	}

	JobResult struct {
		ID       string `json:"id"`
		Response string `json:"body"`
	}
)

func correlateRequest(j *Job) string      { return j.ID }
func correlateResult(r *JobResult) string { return r.ID }
```

### Implement the Batch Processor

```go
type RemoteProcessor struct{}

func (*RemoteProcessor) ProcessJobs(jobs []*Job) ([]*JobResult, error) {
	request, err := json.Marshal(jobs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jobs: %w", err)
	}

	response := request // Send the jobs downstream for processing and retrieve the results.

	var results []*JobResult
	err = json.Unmarshal(response, &results)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal job results: %w", err)
	}

	return results, nil
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
		if result, err := batcher.ExecuteJob(ctx, &Job{ID: "1", Request: "Hello, world"}); err == nil {
			fmt.Println(result.Response)
		}
	}()

	// Shut down
	wg.Wait()
	batcher.Shutdown()
```

## Links

- [Example project calling AWS Lambda](https://github.com/fillmore-labs/microbatch-lambda)
