# Dockaudit 🐳🔍

**Dockaudit** is a production-grade diagnostic and optimization CLI tool for Docker environments. 

It moves beyond basic `docker system prune` wrappers by building a complete **Dependency Graph** of your Docker resources. Dockaudit answers the critical questions: *What is safe to delete? Why does this exist? What created it? What will break if I remove this?*

---

## 🌟 Key Features

* **Actionable Diagnostics (`doctor`)**: Identify zombie resources, orphaned volumes, and build cache waste instantly.
* **Risk Categories**: Replaces blind deletion with safe, human-readable risk assessments (`SAFE`, `LIKELY SAFE`, `REVIEW`, `RISKY`, `DANGEROUS`).
* **Dependency Graph (`explain`)**: Full provenance tracking. Know exactly which Compose Project spawned a volume, and which containers still rely on it.
* **CI/CD Native**: Run `dockaudit doctor --ci` in your automation pipelines to flag environments with high resource waste.
* **Dry-Run by Default**: Safe to use. No resources are ever deleted without an explicit, interactive confirmation using the `clean` command.

---

## 🚀 Installation

*Pre-built binaries and Homebrew taps coming soon.*

To install from source, ensure you have Go 1.24+ installed:

```bash
git clone https://github.com/dockaudit/dockaudit.git
cd dockaudit
go build -o dockaudit
sudo mv dockaudit /usr/local/bin/
```

---

## 🛠️ Usage Guide

### 1. The Hero Command: `doctor`
Get a high-level overview of your environment's health and potential disk recovery.

```bash
$ dockaudit doctor

Docker Doctor Report

✓ Daemon healthy
⚠ 12 volumes need review
⚠ 9 unused/dangling images
⚠ Build cache consuming 4.20 GB

Recovery Potential: 18.70 GB
```

#### CI/CD Automation
Output JSON suitable for pre-commit hooks or ephemeral CI agents:
```bash
$ dockaudit doctor --ci
```

### 2. Provenance Tracking: `explain`
Ask Dockaudit why a specific resource exists. (WIP)

```bash
$ dockaudit explain volume postgres-data

Volume: postgres-data
Risk: DANGEROUS
Reason: Referenced by running postgres container
```

### 3. Cleanup: `plan` & `clean`
Generate a safe cleanup plan based on graph analysis, and execute it interactively. (WIP)

```bash
$ dockaudit plan
$ dockaudit clean
```

---

## 🧠 How It Works

1. **Discovery**: Fetches Containers, Images, Volumes, Networks, and Build Caches via the local Docker SDK.
2. **Graph Construction**: Builds a General Directed Graph connecting resources based on relationships (`Mounts`, `CreatedFrom`, `ConnectedTo`, `BelongsToProject`).
3. **Risk Analysis**: Traverses the graph to assign risk categories. A volume without incoming container edges is `SAFE`. A volume attached to a running container is `DANGEROUS`.

---

## 📁 Architecture
- `cmd/`: Cobra CLI commands.
- `internal/docker/discovery`: Docker SDK integration.
- `internal/graph`: Directed graph models and traversal logic.
- `internal/planner`: Risk rules engine and cleanup plan generator.

---

## 🔒 Security & Privacy

Dockaudit operates **100% locally**. It never connects to external services, sends telemetry, or auto-prunes resources without explicit confirmation. It only talks to your local Docker socket.
