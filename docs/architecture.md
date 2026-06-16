# Architecture & Code Structure

Dockaudit is designed to be highly modular, testable, and resilient to the chaotic reality of Docker environments. 

Rather than writing a massive script that calls `docker system prune`, we engineered a multi-stage pipeline: **Discovery ➡️ Graph Building ➡️ Planning ➡️ Execution**. This decoupling allows us to reason about resources mathematically before touching anything on disk.

Here is a deep dive into the architecture and the reasoning behind our design decisions.

---

## 🏗️ Directory Layout

We adhere to standard Go project layouts to keep the binary footprint small and the domain logic isolated.

```text
dockaudit/
├── cmd/                          # Cobra CLI commands (the entrypoints)
├── internal/
│   ├── docker/discovery/         # Fetches raw data from the Docker API
│   ├── graph/model/              # The core General Directed Graph structs
│   ├── analyzer/relationships/   # Wires up raw Docker data into Graph Edges
│   ├── planner/                  # Rules engine that generates Safe/Risky categorizations
│   ├── cleanup/                  # Safely executes deletions via Docker API
│   ├── report/                   # Formats output for JSON/HTML/Markdown
│   └── trends/                   # Handles local snapshot storage for time-travel tracking
```

---

## 🧠 Core Design Decisions

### 1. The General Directed Graph (Not a DAG)

When you think of dependencies, you usually think of a Directed Acyclic Graph (DAG). *Image A creates Container B creates Volume C.*

**Why we didn't use a DAG:**
Docker resources often form many-to-many relationships or even circular logical relationships. For example, `Container A` and `Container B` might both connect to `Network X`. A `docker-compose` project owns multiple networks, volumes, and containers. 

To handle this, we implemented a **General Directed Graph**.
- `Nodes` represent discrete entities (Containers, Images, Volumes, Compose Projects).
- `Edges` represent relationships (`Mounts`, `CreatedFrom`, `ConnectedTo`, `BelongsToProject`).

This allows us to write traversal functions like `FindIncomingEdges`. If a Volume has *any* incoming edge from a running container, we instantly know it is `DANGEROUS` to delete.

### 2. Strong Typing Over Interfaces

Early versions of resource tracking often rely on generic `interface{}` types for raw data dumps.

**Why we used embedded structs:**
Using `interface{}` forces endless type assertions and makes the codebase fragile. Instead, we created a `BaseResource` struct containing standard fields (ID, Name, Size, CreatedAt).

```go
type BaseResource struct { ... }

type VolumeNode struct {
    BaseResource
    Attached bool
}
```

By embedding `BaseResource` into specific node types (`VolumeNode`, `ContainerNode`, `ComposeProjectNode`), the planner can cleanly iterate over the graph, access universal fields (like SizeBytes for calculating recovery totals), and then switch on the specific type to apply granular rules.

### 3. Decoupling the Planner from the Cleaner

It is tempting to write logic that looks like: *Find Unused Volume ➡️ Delete It.*

**Why we split them:**
We explicitly separated the `internal/planner` from `internal/cleanup`. The Planner operates entirely in memory on the Graph. It generates a `Plan` consisting of `Action` items with assigned Risk Categories (`SAFE`, `RISKY`, `DANGEROUS`).

This decoupling enables:
1. **Dry-runs by default:** We can print the plan to the screen (`dockaudit plan`) without risking data loss.
2. **Interactive execution:** The `cleaner` just loops over the `SAFE` actions in the plan and prompts the user.
3. **Advanced Reporting:** The `report` package takes the generated plan and dumps it to JSON/HTML, entirely independent of the CLI output.

### 4. Compose Projects as First-Class Citizens

Modern developers don't use `docker run`; they use `docker compose up`. 

If Dockaudit only looked at containers and volumes, it would miss the larger context of the project. We made `ComposeProject` a first-class `Node` in the graph. By reading the `com.docker.compose.project` labels provided by the Docker daemon, we generate edges that tie seemingly orphaned networks or volumes back to the project that originally created them. 

This is what makes the `dockaudit explain` command so magical—it understands the *human* intent behind the infrastructure.

---

## 🛡️ The Docker SDK

We strictly use the official `github.com/docker/docker/client` SDK. 

**Why no shell commands?**
Shelling out to `docker ps` or `docker inspect` is brittle, requires complex regex parsing, and breaks across different host OS environments. By pinning to the official SDK and negotiating the API version directly over the local unix socket (`/var/run/docker.sock`), Dockaudit achieves lightning-fast discovery times (< 5 seconds for massive daemon states) with zero external dependencies.
