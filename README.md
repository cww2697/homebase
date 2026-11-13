# Homebase

**Homebase** is a modern, responsive, web-based homelab dashboard and application launcher inspired by [Homepage](https://gethomepage.dev/). It provides comprehensive service monitoring and bookmarks, combined with a built-in **live WYSIWYG editor**, **Proxmox VE & TrueNAS API key integrations**, **dynamic tab/page management**, **persistent Docker configuration storage**, and **Git synchronization** (e.g. GitHub/GitLab).

---

## Key Features

1. **WYSIWYG In-Place Layout Editor**
   - Click **Edit Dashboard** to visually customize widgets without editing YAML or JSON files.
   - Resize widgets dynamically (1, 2, 3, or 4 columns wide, multi-row heights).
   - Reorder widgets in real time with immediate visual feedback.
   - Add new widgets from a built-in catalog of templates.
   - Configure widget properties (titles, URLs, API keys, icons, tags) directly in intuitive modal dialogs.

2. **Pages & Tabs Management**
   - Create, rename, reorder, and delete dashboard tabs (e.g., *Home*, *Infrastructure*, *Services*, *Media*, *Network*).
   - Customize icons for each tab.

3. **Homelab Integrations via Secure Proxy**
   - **Proxmox VE**: Live CPU, memory, root disk usage, and QEMU VM / LXC container counts using standard Proxmox API tokens (`PVEAPIToken=USER@REALM!TOKENID=UUID`). Supports self-signed SSL certificates.
   - **TrueNAS (SCALE & CORE)**: Real-time pool health status (ONLINE, DEGRADED), capacity utilization gauge, alert count, and uptime via TrueNAS API keys (`Bearer <key>`).
   - **Host & System Monitoring**: Host CPU cores, Go runtime memory allocation, goroutines, and application uptime.
   - **Docker Containers**: Running, stopped, and total container metrics with status badges when `/var/run/docker.sock` is mounted.
   - **Local Weather**: Live temperature, weather condition icons, wind speed, and humidity via Open-Meteo (no paid API key required).
   - **Web Search**: Multi-engine search widget supporting DuckDuckGo, Google, Bing, SearXNG, Reddit, and Wikipedia.
   - **Service Health Pings**: Automatic latency and reachability checks for your bookmarked services.
   - **Notes & Runbooks**: In-dashboard scratchpads and server documentation cards.
   - **IFrames / Embeds**: Embed dashboards from Grafana, Pi-hole, AdGuard Home, or Home Assistant.

4. **Reliable Docker & Volume Persistence**
   - Stores all configuration atomically in `config.json` inside `/app/data`.
   - Your layout, API tokens, and tabs survive container restarts and image updates.
   - Config export and import in JSON format right from the footer.

5. **Git Synchronization (GitHub & GitLab)**
   - Link any Git repository to version-control your dashboard configuration.
   - Push updated configurations directly from the UI or enable **Auto-push on Save**.
   - Pull down configuration on demand or configure **Auto-pull on Startup** to automatically seed newly deployed containers from GitHub.

6. **Modern, Responsive Design & Theming**
   - Clean glassmorphism styling with fast render times.
   - Built-in color themes: **Dark Slate**, **Cyberpunk Neon**, **Nord Frost**, **Deep Slate**, **Monokai Pro**, and **Clean Light**.
   - Offline-ready with self-contained Lucide SVG icons (no external CDNs needed).

---

## Quick Start with Docker

### Using Docker Compose (Recommended)

1. Clone or download this repository.
2. Launch the container:
   ```bash
   docker compose up -d
   ```
3. Open `http://localhost:8080` in your web browser.

Configuration data is saved in `./data/config.json` on your host machine.

### Using Docker CLI

```bash
docker run -d \
  --name homebase \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --restart unless-stopped \
  homebase:latest
```

---

## Running Natively (Go)

Ensure you have **Go 1.21+** installed:

```bash
# Build Homebase
go build -o homebase .

# Run Homebase
./homebase -port=8080 -data=./data
```

---

## Configuring Integrations

### Proxmox VE
1. In Proxmox VE, navigate to **Datacenter** &rarr; **Permissions** &rarr; **API Tokens**.
2. Create an API token (e.g., `root@pam!homebase`).
3. In Homebase, toggle **Edit Dashboard**, click the settings icon on the Proxmox widget, and paste your Proxmox URL and token.
4. If your Proxmox installation uses self-signed certificates, ensure **Ignore SSL Certificate warnings** is checked.

### TrueNAS
1. In TrueNAS, navigate to the top-right user menu &rarr; **API Keys** &rarr; **Add**.
2. Copy the generated API key.
3. In Homebase, open the TrueNAS widget settings, enter your TrueNAS URL (e.g. `https://truenas.local`), paste the API key, and specify your primary pool name (e.g. `tank`).

### Git Synchronization
1. Click the **Git branch icon** in the top navigation bar.
2. Enable Git Sync and provide:
   - **Repository URL**: e.g., `https://github.com/yourusername/homelab-dashboard-config.git`
   - **Branch**: e.g., `main`
   - **Personal Access Token**: GitHub/GitLab token with repository read/write permissions.
3. Use **Push to Git** to back up your current layout, or **Pull from Git** to restore it.

---

## Project Architecture

```
Homebase/
├── Dockerfile                  # Multi-stage production container build
├── docker-compose.yml          # Ready-to-deploy Compose configuration
├── go.mod                      # Go module definition
├── main.go                     # Application entrypoint & graceful shutdown
├── internal/
│   ├── config/                 # Atomic config manager & default schemas
│   ├── git/                    # Git CLI & HTTP sync engine
│   ├── handlers/               # REST API & static file serving
│   └── integrations/           # Proxmox, TrueNAS, System, Docker, Weather, Ping
└── web/                        # Embedded web assets
    ├── embed.go                # Go embed.FS filesystem wrapper
    ├── index.html              # Responsive dashboard shell & modal dialogs
    ├── css/style.css           # Glassmorphic styles & color themes
    └── js/
        ├── icons.js            # Inlined Lucide SVG icons library
        ├── api.js              # REST client
        ├── widgets.js          # Live widget renderers
        ├── editor.js           # WYSIWYG editor engine
        └── app.js              # State manager & polling controller
```

---

## REST API Reference

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/health` | `GET` | Health check probe |
| `/api/config` | `GET` | Retrieve the active configuration |
| `/api/config` | `POST` | Update and atomically persist configuration |
| `/api/config/reset` | `POST` | Reset dashboard layout to defaults |
| `/api/git/sync` | `POST` | Trigger a Git pull or push action |
| `/api/proxy/proxmox` | `POST` | Proxy metrics query to Proxmox VE |
| `/api/proxy/truenas` | `POST` | Proxy pool query to TrueNAS |
| `/api/proxy/ping` | `POST` | Service reachability and latency check |
| `/api/proxy/docker` | `POST` | Query Docker engine for container status |
| `/api/system/stats` | `GET` | Host system resource metrics |
| `/api/weather` | `GET` | Current meteorological forecast |
