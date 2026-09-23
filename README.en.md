# Money Graph — Граф денег

[Русский](README.md) · [Қазақша (KZ)](README.kz.md) · **English (EN)**

**Explainable analysis of bank transfer networks for financial monitoring.**
A **Zzz Technology** project for the **HackAlem AI** hackathon.

Money Graph helps analysts decide which nodes in a transaction network to review
first: who collects funds, passes them on, distributes them among recipients, or
connects groups. Each client receives a role, a review priority, and an explanation
that can be checked against the underlying transfers.

**The system produces hypotheses for review, not evidence of wrongdoing.**

## Implemented features

- Load three Parquet files and validate aggregated edges against individual
  transactions by client pair, transfer count, and total amount.
- Compute incoming/outgoing connections and amounts, PageRank, HITS,
  betweenness, transit share, links to traversal starting nodes (`seed`),
  and temporal activity features.
- Assign one of six roles, a `role_score`, a cluster, a `priority_score`,
  and a textual explanation (`evidence`).
- Run Louvain clustering, find cycles of up to five nodes and repeated
  A → B → C routes, and estimate the effects of removing priority nodes.
- Explore a directed graph in React: full gid or prefix search, role and cluster
  filters, top 30, client cards, navigation to neighbors, one/two-hop neighborhoods,
  and card copying.
- Export three CSV files and `graph.json`; provide an output validator,
  an HTTP API, and Swagger documentation.
- Optional AI features: node summaries, cluster hypotheses, and answers to
  questions using graph-reading tools.

| Role | Interpretation |
|---|---|
| `consolidator` | Collects funds from multiple payers and retains a substantial share |
| `transit` | Passes on an amount comparable to the amount received |
| `distributor` | Distributes funds among many recipients |
| `terminal` | Final recipient within the observed dataset |
| `coordinator` | Connects groups and has high betweenness |
| `peripheral` | Insufficient signals to assign any of the other roles |

![Graph analysis interface](docs/ui.png)

## How it works

1. The pipeline reads `data/nodes.parquet`, `edges.parquet`, and `transactions.parquet`.
2. It builds the graph and computes metrics, clusters, and transfer patterns.
3. Rules assign roles; a formula based on metric percentile ranks determines
   review priority. Results are saved to `out/`.
4. The web server computes a shared analysis result in memory and exposes it through an API.
5. An analyst opens the top 30 or enters a client's gid, inspects the card,
   transfer amounts and directions, navigates to counterparties, and expands the neighborhood.
6. If needed, the analyst requests a text summary or asks the assistant a question.

Roles and priorities are computed locally. The LLM receives already calculated
facts and **does not assign** roles, clusters, or priorities.

Clustering uses an undirected projection weighted by transfer amounts and a fixed
random seed. Priority considers incoming volume, payer count, upstream seed nodes,
PageRank, betweenness, and role. Seed nodes and nodes at the traversal boundary
receive downward adjustments. See the [methodology](docs/methodology.md)
(in Russian) for formulas, thresholds, and interpretations.

## Technologies

| Component | Implementation |
|---|---|
| Backend and CLI | Go; module Go version `1.25.7` |
| HTTP, dependency injection, configuration | Fiber v2, Uber FX, Viper, godotenv, zap, validator |
| Data and graph algorithms | `parquet-go`, Gonum, custom metric calculations and rules |
| Frontend | JavaScript, React 19, Vite 7, Cytoscape.js 3.30.4, CSS |
| API documentation | swag / fiber-swagger |
| AI | OpenAI Responses API; default model `gpt-5.1` in `config/base.yaml` |
| Build and runtime | Make, Docker, Docker Compose; Node.js for the React build stage |
| Storage | Parquet, CSV, JSON; analysis results in memory, no database |

The model and API base URL are configured through `LLM_MODEL` and `LLM_BASE_URL`.
Cytoscape and React are bundled locally; the browser does not need external CDNs.

## Architecture

```mermaid
flowchart LR
    D[Parquet files] --> L[Loading and validation]
    L --> M[Graph and metrics]
    M --> C[Clusters and patterns]
    C --> R[Roles and priorities]
    R --> E[CSV and JSON export]
    R --> A[Analysis result in memory]
    A --> API[Fiber API]
    API --> UI[React and Cytoscape]
    E -. graph.json fallback .-> UI
    A --> S[Assistant service]
    S <--> K[JSON cache]
    S -. with API key .-> O[OpenAI Responses API]
    S --> API
```

Docker runs **one `app` service**. React is built in a separate image stage;
Go then serves the static build and API on port 8080. This setup requires no
separate Nginx, Node server, or Postgres instance.

| Directory | Purpose |
|---|---|
| `cmd/pipeline`, `cmd/check`, `cmd/web`, `cmd/ask` | Analysis, output validation, web server, assistant CLI |
| `internal/app`, `internal/config` | Dependency wiring and configuration |
| `internal/repo/dataset` | Parquet loading and validation |
| `internal/data/models`, `internal/data/graph` | Domain models, graph, and indexes |
| `internal/services/analysis` | Metrics, roles, clusters, priorities, and patterns |
| `internal/services/pipeline`, `export`, `graph` | Orchestration, export, and result access |
| `internal/services/assistant`, `hypotheses`, `pkg/llm` | AI workflows, API client, and cache |
| `internal/data/dto`, `internal/transport/http` | JSON contracts and HTTP handlers |
| `frontend` | React source; build output in `frontend/dist` |
| `web` | Embedded fallback viewer and test graph |
| `data`, `out`, `docs`, `tests` | Input data, results, documentation, and tests |

## Installation and startup

### 1. Get the project

```bash
git clone https://github.com/BAITC-Hacks/hack-bf176827-zzz-technology.git
cd hack-bf176827-zzz-technology
```

Run commands from the repository root. Internet access is needed for the initial
dependency downloads and Docker build. Once built, core analysis and graph viewing
work locally without an OpenAI key.

### 2. Choose a startup method

**Docker — no local Go or Node.js installation required.** Install Docker
Engine/Desktop and Docker Compose. For Docker Desktop with WSL, enable WSL integration.

```bash
docker compose up --build
# Alternatively, if Make is available:
make docker
```

The image contains the dataset, React build, and three Go binaries. Startup runs
the pipeline, validates the CSV files, and starts the web server. The host's `out/`
directory is mounted into the container for results and the LLM cache.

**Local startup with React.** Requires Go **1.25.7+**, Node.js **22.12+**
(or 24), npm, and Make.

```bash
make demo-full
```

This installs frontend dependencies, builds React, and runs `pipeline → check → web`.
Do not use `-j`: the interface must finish building before the server starts.

To restart with React already built:

```bash
make demo
```

`make demo` does not build React. If `frontend/dist/index.html` is missing,
the server uses the embedded `web/` viewer.

### 3. Open the application

- Interface: **http://localhost:8080**.
- Swagger: **http://localhost:8080/swagger/index.html**.
- Health check: **http://localhost:8080/v1/health**.
- Analysis results: **`out/`**.

The local process runs in the current terminal; stop it with `Ctrl+C`.
For Docker, use `docker compose down` or `make docker-down`.

### Frontend development

Start the API in the first terminal:

```bash
make web
```

In a second terminal:

```bash
cd frontend
npm ci
npm run dev
```

Vite serves http://localhost:5173 and proxies API requests to port 8080.
Set `API_PROXY_TARGET` to use a different backend address.

### Configuration and WSL

Non-secret settings are read from `config/base.yaml` and
`config/<APP_ENVIRONMENT>.yaml`. Main overrides:

| Variable | Default / purpose |
|---|---|
| `APP_PORT` | `8080`, local Go server port |
| `APP_DATA_DIR` | `data`, input directory |
| `APP_OUT_DIR` | `out`, results and cache directory |
| `APP_ENVIRONMENT` | `local`; set to `prod` in Docker |
| `OPENAI_API_KEY` | Optional key for new LLM requests |
| `LLM_MODEL` | `gpt-5.1` |
| `LLM_BASE_URL` | `https://api.openai.com/v1` |

`.env` is optional; `make env` creates it from `.env.example` only if it does not
already exist. Keep the key out of source code. Compose explicitly passes
`OPENAI_API_KEY`; other container overrides require changes to `environment`
and, when changing the port, `ports` in `docker-compose.yaml`.

In WSL, use Linux versions of Node.js and Go rather than Windows npm on a
`\\wsl.localhost\...` path. If `GOROOT` contains an inherited Windows path:

```bash
unset GOROOT
node -p 'process.platform'  # should print linux
command -v node
command -v npm
make demo-full
```

If port 8080 is occupied, stop the previous instance or run locally with
`APP_PORT=8081 make demo-full` and open http://localhost:8081.

## Verification: a walkthrough for the jury

1. Run `make demo-full` or `docker compose up --build`.
   CSV validation should report **2248 unique nodes**.
2. Open the interface. The initial view shows the top 30 with neighbors;
   colors indicate roles, diamonds indicate seed nodes, and arrows show transfer direction.
3. Enter **`100000003115284100`** and press Enter or «Найти» (Find).
   In the included dataset this is a `consolidator`: **8 incoming** and
   **2 outgoing** counterparties, **2,160,500 KZT** received, **517,000 KZT** sent.
   Its card should appear and its connections should be highlighted.
4. Click a neighbor in the card, then «Ego 2 шага» (2-hop ego).
   Check navigation to another node and neighborhood expansion.
5. Try the role and cluster filters. Reset both, then click «Вся сеть» (Full network).
   The included dataset should show **2248 nodes and 3119 edges**.
6. Open `out/nodes_roles.csv` and compare the gid, role, and explanation with
   the interface. Do not convert gid values to floating-point numbers.

More example clients and demonstration questions are in
[docs/demo.md](docs/demo.md) (in Russian).

### API checks

Run these in a second terminal after starting the server:

```bash
curl -f http://localhost:8080/v1/health
curl -f 'http://localhost:8080/v1/top?n=3'
curl -f http://localhost:8080/v1/nodes/100000003115284100
curl -f 'http://localhost:8080/v1/nodes/100000003115284100/ego?depth=2'
curl -f http://localhost:8080/v1/assistant/status
```

Full card metrics are in `node.features`. All JSON gid values are strings because
they exceed JavaScript's exact integer range. Invalid parameters return HTTP 400;
unknown nodes return 404.

### Automated checks

```bash
make pipeline
make check
make test
go build ./...
cd frontend
npm ci
npm test
npm run build
```

`make check` validates node count and uniqueness, allowed roles, scores in [0,1],
non-empty explanations of up to 200 characters, cluster membership, and a sorted
top list with at least 20 entries. It exits with a nonzero code on failure.
Go tests cover analysis determinism, role constraints, graph filters, and HTTP
contracts, among other checks. Frontend tests cover graph operations.

## Data, outputs, and integrations

The repository includes transfers for **July 1–31, 2026**:
**2248 clients, 3119 aggregated directed edges, 4840 transactions, and 81 seed nodes**.
Amounts are in KZT.

| Source | Parquet fields |
|---|---|
| `data/nodes.parquet` | `gid`, `depth`, `is_seed` |
| `data/edges.parquet` | `src`, `dst`, `sum_kzt`, `n_tx`, `depth` |
| `data/transactions.parquet` | `src`, `dst`, `date`, `sum_kzt` |

These are file-based extracts; there is no connection to a banking system or
live transaction stream. The methodology describes the source selection:
four outgoing-transfer hops from seed nodes, with a 5000 KZT threshold.

| Output | Main fields |
|---|---|
| `out/nodes_roles.csv` | `gid,role,role_score,cluster_id,priority_score,evidence` and metrics |
| `out/clusters.csv` | `cluster_id,n_nodes,n_seed,sum_kzt_internal,top_gids,hypothesis` and aggregates |
| `out/top_nodes.csv` | `rank,gid,role,priority_score,why` |
| `out/graph.json` | `nodes,edges,clusters,top,robustness` |
| `out/llm_cache.json` | Cached text answers and hypotheses |

The current dataset produces **88 clusters** and a top list of **30 nodes**.
Role counts: 31 consolidator, 43 coordinator, 16 distributor, 105 transit,
153 terminal, and 1900 peripheral. These counts describe the included dataset;
they are not a measure of model accuracy.

### HTTP API

| Method and path | Purpose |
|---|---|
| `GET /v1/health` | Server availability |
| `GET /v1/graph?role=&cluster=&component=&top=30` | Graph filtering; top N with one-hop neighbors |
| `GET /v1/nodes/{gid}` | Card with metrics and incoming/outgoing counterparties |
| `GET /v1/nodes/{gid}/ego?depth=1` | Neighborhood in both directions, depth 1–2 |
| `GET /v1/top?n=30` | Priority nodes from the computed top list |
| `GET /v1/clusters` | Clusters and hypotheses |
| `GET /v1/search?q=1000&limit=10` | gid prefix search |
| `GET /v1/assistant/status` | LLM configuration status and model |
| `GET /v1/nodes/{gid}/card` | Text summary |
| `POST /v1/assistant` | Question: `{"question":"..."}`; answer with text and gids |
| `GET /graph.json` | Latest export file used as the UI's fallback data source |

For `/v1/graph`, omitting `top` or setting `top=0` selects the entire network.
`/v1/top` is limited to the size of the top list computed at startup.

### Optional AI features

The only external AI integration is the **OpenAI Responses API**. To request
new answers, set `OPENAI_API_KEY` in the environment or `.env` and restart the server.
These requests send graph facts and node identifiers to the external API.

Without a key:

- Analysis, roles, priorities, and the interface work locally.
- Cluster hypotheses are read from the cache or generated from templates.
- Node summaries come from the cache or templates.
- Assistant answers are available if the question is cached. A new question
  returns HTTP 503; the interface hides unavailable AI controls after a 503 response.

The standard pipeline does not request new hypotheses from the model. To enable this explicitly:

```bash
go run ./cmd/pipeline --data data --out out --llm
# CLI node summary; also works without a key:
go run ./cmd/ask --card 100000003115284100
# A new question needs a key unless its answer is cached:
go run ./cmd/ask --q 'Which nodes have the highest priority, and why?'
```

## Current limitations

- Only the provided extract is visible: full client histories, below-threshold
  transfers, and flows beyond the traversal boundary are missing. A fourth-hop
  node with no outgoing edges does not prove that it retained the funds.
- Roles are heuristic. The repository has no ground-truth labels for measuring
  accuracy; `role_score` is not a validated probability of wrongdoing.
- Analysis runs in memory at startup. Production-scale processing, streaming
  updates, and uploading new files through the UI are not implemented.
  The `cmd/check` validator is tied to the hackathon dataset's 2248 nodes.
- Clustering uses an undirected projection. Direction is retained in the transfer
  graph but not in this grouping step.
- Laying out the full network can take noticeable time. The default view uses
  the smaller top-30 subgraph with neighbors; timing depends on the computer and browser.
- The React card displays basic metrics. Full metrics are available in the API's
  `node.features`; not all nested metrics are currently displayed in the UI.
- User authentication and access controls are not implemented.
  The current configuration is intended for local demonstrations.
- New AI answers need an external API and key. Generated text may contain errors
  and must be checked against the underlying facts. No custom-trained ML model is included.

## Deployed version

**The repository does not specify a public deployment URL.**
The documented demonstration method is a local launch at http://localhost:8080.

Further materials (in Russian): [methodology](docs/methodology.md),
[demo walkthrough](docs/demo.md), [solution diagram](docs/scheme.png).
