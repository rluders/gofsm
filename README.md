# GOFSM

An extensible Finite State Machine (FSM) library for Go, designed for event-driven and distributed systems.

> [!WARNING]
> This library was initially created as a proof of concept (PoC) to solve a specific coordination issue in a distributed system.
> It is **not production-ready** and may contain bugs or limitations that have not yet been addressed.
> Contributions, suggestions, and improvements are **very welcome**!

## 🧩 What Problem Does It Solve?

In distributed systems, managing workflows and transitions across multiple services can become complex—especially when coordination, concurrency, and external events are involved.

`gofsm` aims to provide a decoupled, pluggable, and testable FSM core that allows:

- Safe state transitions under concurrency (with Redis locking support)
- Reactive behavior driven by events (from Kafka, HTTP, etc.)
- Clean separation between FSM logic and infrastructure concerns

It is ideal for scenarios such as:

- Orchestrating Kubernetes jobs
- Reacting to Kafka or event-stream inputs
- Coordinating microservices through stateful flows

## ✨ Features

- Reusable FSM engine with `OnEnter`, `HandleEvent`, and `OnExit` lifecycle hooks
- Fully decoupled from transport and storage
- Pluggable storage layer with Redis adapter (uses lock + retry)
- Optional Kafka integration using `segmentio/kafka-go`
- Support for `TransitionHook` to notify or trigger side-effects
- Designed for testability and distributed coordination

## 📦 Installation

```bash
go get github.com/rluders/gofsm@latest
```

## 🚀 Basic Usage

```go
engine, err := fsm.NewFSM([]fsm.State{
    &MyInitialState{},
    &MyFinalState{},
}, fsm.WithStateStorage(myStorage))

// Triggering an event
err = engine.Trigger(ctx, "entity-id", fsm.NewBasicEvent("MyEvent", nil))
```

## 📄 License

MIT

