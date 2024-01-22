# Micro Batcher

[![Go Reference](https://pkg.go.dev/badge/fillmore-labs.com/microbatch.svg)](https://pkg.go.dev/fillmore-labs.com/microbatch)
[![Build Status](https://badge.buildkite.com/1d68e28b14ecbbd4e4066e61c25f81ef08a8237615f5d03a6a.svg)](https://buildkite.com/fillmore-labs/microbatch)
[![Test Coverage](https://codecov.io/gh/fillmore-labs/microbatch/graph/badge.svg?token=Sh0xNVeFCd)](https://codecov.io/gh/fillmore-labs/microbatch)
[![Maintainability](https://api.codeclimate.com/v1/badges/2ba503a6a37cfc77951c/maintainability)](https://codeclimate.com/github/fillmore-labs/microbatch/maintainability)
[![Go Report Card](https://goreportcard.com/badge/fillmore-labs.com/microbatch)](https://goreportcard.com/report/fillmore-labs.com/microbatch)
[![License](https://img.shields.io/github/license/fillmore-labs/microbatch)](https://github.com/fillmore-labs/microbatch/blob/main/LICENSE)

Micro-batching is a technique often used in stream processing to achieve near real-time computation while reducing the
overhead compared to single record processing.
It balances latency versus throughput and enables simplified parallelization while optimizing resource utilization.

See also the definition in the [Hazelcast Glossary](https://hazelcast.com/glossary/micro-batch-processing/) and
explanation by [Jakob Jenkov](https://jenkov.com/tutorials/java-performance/micro-batching.html).
Popular examples are
[Spark Structured Streaming](https://spark.apache.org/docs/latest/structured-streaming-programming-guide.html#overview)
and [Apache Kafka](https://kafka.apache.org/documentation/#upgrade_11_message_format).
It is also used in other contexts, like the [Facebook DataLoader](https://github.com/graphql/dataloader#batching).

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

## Design

The package is designed to handle request batching efficiently, with a focus on code testability and modular
architecture.
The codebase is organized into three packages: the public `microbatch.Batcher` structure and two internal helpers,
`processor.Processor` and `collector.Collector`.

### Motivation

The primary design goal is to enhance code testability, enabling unit testing of individual components in isolation,
with less emphasis on immediate performance gains.

While an alternative approach might involve constructing the correlation map during batch collection for performance
reasons, the current design prioritizes testability and separation of concerns.
In this context, the collector remains independent of correlation IDs, focusing solely on batch size and timing
decisions.
The responsibility of correlating requests and responses is encapsulated within the processor, contributing to a
cleaner and more modular architecture.

### Component Description

By maintaining a modular structure and addressing concurrency issues, the codebase is designed to achieve good
testability while still maintaining high performance and offering flexibility for future optimizations.
The deliberate use of channels and immutability contributes to a more straightforward and reliable execution.

#### Public Interface (`microbatch.Batcher`)

The public interface is the entry point for users interacting with the batching functionality.
It is designed to be thread-safe, allowing safe invocation from any goroutine and simplifying usage.
It communicates with the collector exclusively using channels.
This deliberate choice simplifies synchronization concerns within the collector, contributing to a cleaner design and
simplified reasoning about potential synchronization issues.

#### Collector (`collector.Collector`)

The collector is responsible for managing queued requests and initiating batch processing.
It operates in a single goroutine, eliminating the need for locks.
This design choice simplifies the execution flow and minimizes potential contention issues.
The collector maintains an array of queued requests and, when a complete batch is formed or a maximum collection time
is reached, spawns a processor.
The processor takes ownership of the queued requests, correlating individual requests and responses.

#### Processor (`processor.Processor`)

The processor wraps the user-supplied processor, handling the correlation of requests and responses.
Once constructed, the fields are accessed read-only, ensuring immutability.
This enables multiple processors to operate in parallel without conflicts.
By encapsulating the responsibility of correlation, the processor contributes to a modular and clean architecture,
promoting separation of concerns.

## Links

- [Example project calling AWS Lambda](https://github.com/fillmore-labs/microbatch-lambda)
