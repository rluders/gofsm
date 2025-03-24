# FSM Scheduler Example

This directory `examples/scheduler-example` contains a full working example of a distributed application using the [gofsm](https://github.com/rluders/gofsm) library with Redis, Kafka and multiple concurrent instances. It includes two microservices:

- **dispatcher**: exposes a REST API to create scans.
- **scheduler**: consumes Kafka events and coordinates scan and scan job state transitions.

## Requirements

- Docker + Docker Compose or Podman
- Go 1.21+
- Vegeta (optional, for load testing)

## Running the full environment

```bash
cd examples/scheduler-example
podman compose up --build # or docker compose up --build
```

This will start:

- Kafka + Zookeeper
- Redis
- 2 `scheduler` instances
- 1 `dispatcher` instance

## Creating a scan manually

```bash
curl -X POST http://localhost:8080/scans \
  -H "Content-Type: application/json" \
  -d '{"job_count": 5}'
```

You will receive a response like:

```json
{
  "scan_id": "5a12...",
  "status": "pending"
}
```

## Checking scan status

```bash
curl http://localhost:8080/scans/5a12...
```

## Load testing with Vegeta

Create a file named `payload.json`:

```json
{"job_count": 10}
```

Then run:

```bash
echo 'POST http://localhost:8080/scans' | vegeta attack \
  -rate=100 -duration=10s \
  -header="Content-Type: application/json" \
  -body=payload.json \
  | tee results.bin | vegeta report
```

## Expected behavior

- Logs from both `fsm_scheduler_1` and `fsm_scheduler_2` handling scan messages.
- Redis-based locking and retries managing race conditions.
- No state conflicts or processing duplication.

## Notes

- The `dispatcher` sends scan events to Kafka.
- The `scheduler` listens to the `scans` topic and controls the FSM states with Redis locking.
- Each `Scan` contains multiple `ScanJobs`. Transition to `completed` only happens when all jobs are completed.

## License

MIT

