/**
 * Homebase Widget Renderers
 * Builds DOM elements and live data feeds for every widget type
 */

const Widgets = {
  render(widget, isEditing = false) {
    const card = document.createElement('div');
    card.id = `widget-${widget.id}`;
    card.className = `widget-card col-span-${widget.colSpan || 2} row-span-${widget.rowSpan || 1}`;
    card.setAttribute('data-widget-id', widget.id);

    // Edit Bar (visible in WYSIWYG edit mode)
    const editBar = this.createEditBar(widget);
    card.appendChild(editBar);

    // Content container
    const content = document.createElement('div');
    content.className = 'widget-inner-content';

    switch (widget.type) {
      case 'service':
        this.renderService(content, widget);
        break;
      case 'proxmox':
        this.renderProxmox(content, widget);
        break;
      case 'truenas':
        this.renderTrueNAS(content, widget);
        break;
      case 'system':
        this.renderSystem(content, widget);
        break;
      case 'docker':
        this.renderDocker(content, widget);
        break;
      case 'weather':
        this.renderWeather(content, widget);
        break;
      case 'clock':
        this.renderClock(content, widget);
        break;
      case 'search':
        this.renderSearch(content, widget);
        break;
      case 'notes':
        this.renderNotes(content, widget);
        break;
      case 'iframe':
        this.renderIFrame(content, widget);
        break;
      default:
        this.renderGeneric(content, widget);
        break;
    }

    card.appendChild(content);
    return card;
  },

  createEditBar(widget) {
    const bar = document.createElement('div');
    bar.className = 'widget-edit-bar';

    const titleSpan = document.createElement('span');
    titleSpan.className = 'font-medium';
    titleSpan.textContent = widget.title || widget.type;

    const actions = document.createElement('div');
    actions.className = 'widget-edit-actions';

    // Move Left button
    const moveLeft = document.createElement('button');
    moveLeft.className = 'edit-btn';
    moveLeft.title = 'Move Left';
    moveLeft.innerHTML = Icons.get('arrow-left', 14);
    moveLeft.onclick = (e) => {
      e.stopPropagation();
      Editor.moveWidget(widget.id, -1);
    };

    // Move Right button
    const moveRight = document.createElement('button');
    moveRight.className = 'edit-btn';
    moveRight.title = 'Move Right';
    moveRight.innerHTML = Icons.get('arrow-right', 14);
    moveRight.onclick = (e) => {
      e.stopPropagation();
      Editor.moveWidget(widget.id, 1);
    };

    // Size dropdown
    const sizeSelect = document.createElement('select');
    sizeSelect.className = 'size-select';
    sizeSelect.title = 'Resize Widget';
    sizeSelect.innerHTML = `
      <option value="1"${widget.colSpan === 1 ? ' selected' : ''}>1 Col</option>
      <option value="2"${widget.colSpan === 2 ? ' selected' : ''}>2 Col</option>
      <option value="3"${widget.colSpan === 3 ? ' selected' : ''}>3 Col</option>
      <option value="4"${widget.colSpan === 4 ? ' selected' : ''}>4 Col (Full)</option>
    `;
    sizeSelect.onchange = (e) => {
      Editor.resizeWidget(widget.id, parseInt(e.target.value, 10));
    };

    // Edit config button
    const editBtn = document.createElement('button');
    editBtn.className = 'edit-btn';
    editBtn.title = 'Configure Widget';
    editBtn.innerHTML = Icons.get('sliders', 14);
    editBtn.onclick = (e) => {
      e.stopPropagation();
      Editor.openWidgetModal(widget.id);
    };

    // Delete button
    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'edit-btn danger';
    deleteBtn.title = 'Delete Widget';
    deleteBtn.innerHTML = Icons.get('trash-2', 14);
    deleteBtn.onclick = (e) => {
      e.stopPropagation();
      Editor.deleteWidget(widget.id);
    };

    actions.append(moveLeft, moveRight, sizeSelect, editBtn, deleteBtn);
    bar.append(titleSpan, actions);
    return bar;
  },

  renderService(container, widget) {
    const props = widget.properties || {};
    const url = props.url || '#';
    const desc = props.description || '';
    const iconName = props.icon || 'server';
    const tag = props.tag || '';

    container.innerHTML = `
      <a href="${url}" target="_blank" rel="noopener noreferrer" class="service-widget">
        <div class="service-body">
          <div class="service-icon-box">
            ${Icons.get(iconName, 22)}
          </div>
          <div class="service-details">
            <div class="service-name">${this.escapeHTML(widget.title)}</div>
            ${desc ? `<div class="service-desc">${this.escapeHTML(desc)}</div>` : ''}
          </div>
        </div>
        <div class="service-footer">
          <span class="service-tag">${tag ? this.escapeHTML(tag) : 'Service'}</span>
          <span class="service-ping-badge" id="ping-${widget.id}">
            <span class="status-dot"></span> Checking...
          </span>
        </div>
      </a>
    `;

    // Trigger ping check
    const pingTarget = props.pingUrl || props.url;
    if (pingTarget) {
      this.checkServiceHealth(widget.id, pingTarget);
    }
  },

  async checkServiceHealth(widgetId, url) {
    try {
      const res = await API.checkPing(url);
      const badge = document.getElementById(`ping-${widgetId}`);
      if (badge) {
        if (res.online) {
          badge.className = 'service-ping-badge';
          badge.innerHTML = `<span class="status-dot online"></span> ${res.latencyMs}ms`;
        } else {
          badge.className = 'service-ping-badge offline';
          badge.innerHTML = `<span class="status-dot"></span> Offline`;
        }
      }
    } catch (_) {
      const badge = document.getElementById(`ping-${widgetId}`);
      if (badge) {
        badge.className = 'service-ping-badge offline';
        badge.innerHTML = `<span class="status-dot"></span> Unreachable`;
      }
    }
  },

  renderProxmox(container, widget) {
    container.innerHTML = `
      <div class="widget-header">
        <div class="widget-title-group">
          <span class="widget-icon">${Icons.get('server', 18)}</span>
          <span class="widget-title">${this.escapeHTML(widget.title)}</span>
        </div>
        <span class="widget-tag" id="pve-status-${widget.id}">Connecting...</span>
      </div>
      <div class="metrics-row" id="pve-metrics-${widget.id}">
        <div class="metric-stat">
          <span class="stat-label">CPU Usage</span>
          <span class="stat-val" id="pve-cpu-${widget.id}">--%</span>
          <div class="progress-bar-bg"><div class="progress-bar-fill" id="pve-cpu-bar-${widget.id}" style="width: 0%"></div></div>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Memory</span>
          <span class="stat-val" id="pve-mem-${widget.id}">--%</span>
          <div class="progress-bar-bg"><div class="progress-bar-fill" id="pve-mem-bar-${widget.id}" style="width: 0%"></div></div>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Virtual Machines</span>
          <span class="stat-val" id="pve-vms-${widget.id}">--</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">LXC Containers</span>
          <span class="stat-val" id="pve-lxc-${widget.id}">--</span>
        </div>
      </div>
    `;

    this.refreshProxmox(widget);
  },

  async refreshProxmox(widget) {
    const props = widget.properties || {};
    try {
      const data = await API.queryProxmox({
        url: props.url || '',
        apiToken: props.apiToken || '',
        node: props.node || 'pve',
        insecureSkipVerify: props.insecureSkipVerify !== false
      });

      const statusEl = document.getElementById(`pve-status-${widget.id}`);
      const cpuEl = document.getElementById(`pve-cpu-${widget.id}`);
      const cpuBar = document.getElementById(`pve-cpu-bar-${widget.id}`);
      const memEl = document.getElementById(`pve-mem-${widget.id}`);
      const memBar = document.getElementById(`pve-mem-bar-${widget.id}`);
      const vmsEl = document.getElementById(`pve-vms-${widget.id}`);
      const lxcEl = document.getElementById(`pve-lxc-${widget.id}`);

      if (data.online) {
        if (statusEl) {
          statusEl.textContent = `Online • ${data.node}`;
          statusEl.style.color = 'var(--accent)';
        }
        if (cpuEl) cpuEl.textContent = `${data.cpuUsage.toFixed(1)}%`;
        if (cpuBar) {
          cpuBar.style.width = `${Math.min(100, data.cpuUsage)}%`;
          cpuBar.className = data.cpuUsage > 85 ? 'progress-bar-fill danger' : (data.cpuUsage > 65 ? 'progress-bar-fill warn' : 'progress-bar-fill');
        }
        if (memEl) memEl.textContent = `${data.memPercent.toFixed(1)}%`;
        if (memBar) {
          memBar.style.width = `${Math.min(100, data.memPercent)}%`;
          memBar.className = data.memPercent > 90 ? 'progress-bar-fill danger' : 'progress-bar-fill';
        }
        if (vmsEl) vmsEl.textContent = data.vmCount;
        if (lxcEl) lxcEl.textContent = data.lxcCount;
      } else {
        if (statusEl) {
          statusEl.textContent = data.status || 'Offline';
          statusEl.style.color = 'var(--danger)';
        }
      }
    } catch (err) {
      const statusEl = document.getElementById(`pve-status-${widget.id}`);
      if (statusEl) {
        statusEl.textContent = 'Unreachable';
        statusEl.style.color = 'var(--danger)';
      }
    }
  },

  renderTrueNAS(container, widget) {
    container.innerHTML = `
      <div class="widget-header">
        <div class="widget-title-group">
          <span class="widget-icon">${Icons.get('hard-drive', 18)}</span>
          <span class="widget-title">${this.escapeHTML(widget.title)}</span>
        </div>
        <span class="widget-tag" id="truenas-status-${widget.id}">Connecting...</span>
      </div>
      <div class="metrics-row" id="truenas-metrics-${widget.id}">
        <div class="metric-stat">
          <span class="stat-label">Pool Health</span>
          <span class="stat-val" id="truenas-health-${widget.id}">--</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Capacity Used</span>
          <span class="stat-val" id="truenas-cap-${widget.id}">--%</span>
          <div class="progress-bar-bg"><div class="progress-bar-fill" id="truenas-cap-bar-${widget.id}" style="width: 0%"></div></div>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Active Alerts</span>
          <span class="stat-val" id="truenas-alerts-${widget.id}">--</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Host Uptime</span>
          <span class="stat-val" id="truenas-uptime-${widget.id}">--</span>
        </div>
      </div>
    `;

    this.refreshTrueNAS(widget);
  },

  async refreshTrueNAS(widget) {
    const props = widget.properties || {};
    try {
      const data = await API.queryTrueNAS({
        url: props.url || '',
        apiKey: props.apiKey || '',
        pool: props.pool || 'tank',
        insecureSkipVerify: props.insecureSkipVerify !== false
      });

      const statusEl = document.getElementById(`truenas-status-${widget.id}`);
      const healthEl = document.getElementById(`truenas-health-${widget.id}`);
      const capEl = document.getElementById(`truenas-cap-${widget.id}`);
      const capBar = document.getElementById(`truenas-cap-bar-${widget.id}`);
      const alertsEl = document.getElementById(`truenas-alerts-${widget.id}`);
      const uptimeEl = document.getElementById(`truenas-uptime-${widget.id}`);

      if (data.online) {
        if (statusEl) {
          statusEl.textContent = `Online • ${data.hostname || data.poolName}`;
          statusEl.style.color = 'var(--accent)';
        }
        if (healthEl) {
          healthEl.textContent = data.poolHealth || 'ONLINE';
          healthEl.style.color = data.poolHealth === 'ONLINE' ? 'var(--accent)' : 'var(--danger)';
        }
        if (capEl) capEl.textContent = `${data.usedPct.toFixed(1)}%`;
        if (capBar) {
          capBar.style.width = `${Math.min(100, data.usedPct)}%`;
          capBar.className = data.usedPct > 85 ? 'progress-bar-fill danger' : 'progress-bar-fill';
        }
        if (alertsEl) {
          alertsEl.textContent = data.alertCount;
          alertsEl.style.color = data.alertCount > 0 ? 'var(--warning)' : 'var(--text-main)';
        }
        if (uptimeEl) uptimeEl.textContent = data.uptime || '--';
      } else {
        if (statusEl) {
          statusEl.textContent = data.status || 'Offline';
          statusEl.style.color = 'var(--danger)';
        }
      }
    } catch (_) {
      const statusEl = document.getElementById(`truenas-status-${widget.id}`);
      if (statusEl) {
        statusEl.textContent = 'Unreachable';
        statusEl.style.color = 'var(--danger)';
      }
    }
  },

  renderSystem(container, widget) {
    container.innerHTML = `
      <div class="widget-header">
        <div class="widget-title-group">
          <span class="widget-icon">${Icons.get('cpu', 18)}</span>
          <span class="widget-title">${this.escapeHTML(widget.title)}</span>
        </div>
        <span class="widget-tag" id="sys-platform-${widget.id}">System</span>
      </div>
      <div class="metrics-row">
        <div class="metric-stat">
          <span class="stat-label">CPU Cores</span>
          <span class="stat-val" id="sys-cores-${widget.id}">--</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Memory Alloc</span>
          <span class="stat-val" id="sys-mem-${widget.id}">-- MB</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Goroutines</span>
          <span class="stat-val" id="sys-goroutines-${widget.id}">--</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">App Uptime</span>
          <span class="stat-val" id="sys-uptime-${widget.id}">--</span>
        </div>
      </div>
    `;

    this.refreshSystem(widget);
  },

  async refreshSystem(widget) {
    try {
      const stats = await API.getSystemStats();
      const platEl = document.getElementById(`sys-platform-${widget.id}`);
      const coresEl = document.getElementById(`sys-cores-${widget.id}`);
      const memEl = document.getElementById(`sys-mem-${widget.id}`);
      const goEl = document.getElementById(`sys-goroutines-${widget.id}`);
      const uptimeEl = document.getElementById(`sys-uptime-${widget.id}`);

      if (platEl) platEl.textContent = `${stats.os}/${stats.arch}`;
      if (coresEl) coresEl.textContent = stats.numCpu;
      if (memEl) memEl.textContent = `${(stats.memoryAlloc / 1024 / 1024).toFixed(1)} MB`;
      if (goEl) goEl.textContent = stats.numGoroutine;

      if (uptimeEl) {
        const hrs = Math.floor(stats.uptimeSeconds / 3600);
        const mins = Math.floor((stats.uptimeSeconds % 3600) / 60);
        uptimeEl.textContent = `${hrs}h ${mins}m`;
      }
    } catch (_) {}
  },

  renderDocker(container, widget) {
    container.innerHTML = `
      <div class="widget-header">
        <div class="widget-title-group">
          <span class="widget-icon">${Icons.get('box', 18)}</span>
          <span class="widget-title">${this.escapeHTML(widget.title)}</span>
        </div>
        <span class="widget-tag" id="docker-status-${widget.id}">Docker</span>
      </div>
      <div class="metrics-row" style="margin-bottom: 0.75rem;">
        <div class="metric-stat">
          <span class="stat-label">Running</span>
          <span class="stat-val" id="docker-running-${widget.id}" style="color: var(--accent);">--</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Stopped</span>
          <span class="stat-val" id="docker-stopped-${widget.id}">--</span>
        </div>
        <div class="metric-stat">
          <span class="stat-label">Total</span>
          <span class="stat-val" id="docker-total-${widget.id}">--</span>
        </div>
      </div>
      <div class="docker-container-list" id="docker-list-${widget.id}">
        <div class="text-dim" style="font-size: 0.8rem; padding: 0.5rem;">Connecting to Docker socket...</div>
      </div>
    `;

    this.refreshDocker(widget);
  },

  async refreshDocker(widget) {
    const props = widget.properties || {};
    try {
      const data = await API.queryDocker(props.socket || '');
      const statusEl = document.getElementById(`docker-status-${widget.id}`);
      const runningEl = document.getElementById(`docker-running-${widget.id}`);
      const stoppedEl = document.getElementById(`docker-stopped-${widget.id}`);
      const totalEl = document.getElementById(`docker-total-${widget.id}`);
      const listEl = document.getElementById(`docker-list-${widget.id}`);

      if (data.online) {
        if (statusEl) statusEl.textContent = 'Socket Connected';
        if (runningEl) runningEl.textContent = data.running;
        if (stoppedEl) stoppedEl.textContent = data.stopped;
        if (totalEl) totalEl.textContent = data.total;

        if (listEl) {
          if (data.containers.length === 0) {
            listEl.innerHTML = `<div class="text-dim" style="font-size: 0.8rem;">No containers found</div>`;
          } else {
            listEl.innerHTML = data.containers.map(c => `
              <div class="docker-item">
                <span class="docker-item-name">${this.escapeHTML(c.name || c.id)}</span>
                <span class="docker-status-pill ${c.state !== 'running' ? 'stopped' : ''}">${this.escapeHTML(c.state)}</span>
              </div>
            `).join('');
          }
        }
      } else {
        if (statusEl) statusEl.textContent = 'Unavailable';
        if (listEl) {
          listEl.innerHTML = `<div class="text-dim" style="font-size: 0.8rem; color: var(--text-dim);">${this.escapeHTML(data.error || 'Mount /var/run/docker.sock to view containers')}</div>`;
        }
      }
    } catch (_) {}
  },

  renderWeather(container, widget) {
    container.innerHTML = `
      <div class="widget-header">
        <div class="widget-title-group">
          <span class="widget-icon">${Icons.get('cloud', 18)}</span>
          <span class="widget-title">${this.escapeHTML(widget.title)}</span>
        </div>
        <span class="widget-tag" id="weather-city-${widget.id}">Weather</span>
      </div>
      <div class="weather-body">
        <div>
          <div class="weather-temp-large" id="weather-temp-${widget.id}">--°</div>
          <div class="weather-condition" id="weather-cond-${widget.id}">Loading...</div>
        </div>
        <div class="weather-details-col">
          <div id="weather-wind-${widget.id}">Wind: -- km/h</div>
          <div id="weather-humid-${widget.id}">Humidity: --%</div>
        </div>
      </div>
    `;

    this.refreshWeather(widget);
  },

  async refreshWeather(widget) {
    const props = widget.properties || {};
    try {
      const data = await API.getWeather(props.latitude, props.longitude, props.city, props.units);
      const cityEl = document.getElementById(`weather-city-${widget.id}`);
      const tempEl = document.getElementById(`weather-temp-${widget.id}`);
      const condEl = document.getElementById(`weather-cond-${widget.id}`);
      const windEl = document.getElementById(`weather-wind-${widget.id}`);
      const humidEl = document.getElementById(`weather-humid-${widget.id}`);

      if (data.online) {
        if (cityEl) cityEl.textContent = data.city || 'Local';
        if (tempEl) tempEl.textContent = `${Math.round(data.temperature)}${data.unit}`;
        if (condEl) condEl.textContent = data.condition;
        if (windEl) windEl.textContent = `Wind: ${data.windSpeed} km/h`;
        if (humidEl) humidEl.textContent = `Humidity: ${data.humidity}%`;

        // Update topbar weather as well
        const topTemp = document.getElementById('header-weather-temp');
        const topCity = document.getElementById('header-weather-city');
        const topBox = document.getElementById('header-weather');
        if (topTemp && topCity && topBox) {
          topTemp.textContent = `${Math.round(data.temperature)}${data.unit}`;
          topCity.textContent = data.city || '';
          topBox.style.display = 'flex';
        }
      }
    } catch (_) {}
  },

  renderClock(container, widget) {
    container.innerHTML = `
      <div class="clock-widget-body">
        <div class="clock-time-large" id="clock-time-${widget.id}">00:00:00</div>
        <div class="clock-date-large" id="clock-date-${widget.id}">Thursday, January 1, 1970</div>
      </div>
    `;

    const updateClock = () => {
      const now = new Date();
      const timeEl = document.getElementById(`clock-time-${widget.id}`);
      const dateEl = document.getElementById(`clock-date-${widget.id}`);

      if (timeEl) {
        timeEl.textContent = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
      }
      if (dateEl) {
        dateEl.textContent = now.toLocaleDateString([], { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });
      }
    };

    updateClock();
    setInterval(updateClock, 1000);
  },

  renderSearch(container, widget) {
    const props = widget.properties || {};
    const defaultProvider = props.provider || 'duckduckgo';

    container.innerHTML = `
      <form class="search-form" id="search-form-${widget.id}">
        <select class="search-provider-select" id="search-provider-${widget.id}">
          <option value="duckduckgo"${defaultProvider === 'duckduckgo' ? ' selected' : ''}>DuckDuckGo</option>
          <option value="google"${defaultProvider === 'google' ? ' selected' : ''}>Google</option>
          <option value="bing"${defaultProvider === 'bing' ? ' selected' : ''}>Bing</option>
          <option value="searxng"${defaultProvider === 'searxng' ? ' selected' : ''}>SearXNG</option>
          <option value="reddit"${defaultProvider === 'reddit' ? ' selected' : ''}>Reddit</option>
          <option value="wikipedia"${defaultProvider === 'wikipedia' ? ' selected' : ''}>Wikipedia</option>
        </select>
        <input type="text" class="search-input" id="search-query-${widget.id}" placeholder="Search web or type URL..." autocomplete="off">
        <button type="submit" class="btn btn-primary" style="padding: 0.6rem 0.9rem;">
          ${Icons.get('search', 16)}
        </button>
      </form>
    `;

    setTimeout(() => {
      const form = document.getElementById(`search-form-${widget.id}`);
      if (form) {
        form.onsubmit = (e) => {
          e.preventDefault();
          const query = document.getElementById(`search-query-${widget.id}`).value.trim();
          const provider = document.getElementById(`search-provider-${widget.id}`).value;
          if (!query) return;

          let targetURL = `https://duckduckgo.com/?q=${encodeURIComponent(query)}`;
          if (provider === 'google') targetURL = `https://www.google.com/search?q=${encodeURIComponent(query)}`;
          if (provider === 'bing') targetURL = `https://www.bing.com/search?q=${encodeURIComponent(query)}`;
          if (provider === 'searxng') targetURL = `https://searx.be/search?q=${encodeURIComponent(query)}`;
          if (provider === 'reddit') targetURL = `https://www.reddit.com/search/?q=${encodeURIComponent(query)}`;
          if (provider === 'wikipedia') targetURL = `https://en.wikipedia.org/wiki/Special:Search?search=${encodeURIComponent(query)}`;

          window.open(targetURL, '_blank');
        };
      }
    }, 0);
  },

  renderNotes(container, widget) {
    const props = widget.properties || {};
    const content = props.content || 'Click Edit to add notes or server documentation.';

    container.innerHTML = `
      <div class="widget-header">
        <div class="widget-title-group">
          <span class="widget-icon">${Icons.get('file-text', 18)}</span>
          <span class="widget-title">${this.escapeHTML(widget.title)}</span>
        </div>
      </div>
      <div class="notes-content">${this.escapeHTML(content)}</div>
    `;
  },

  renderIFrame(container, widget) {
    const props = widget.properties || {};
    const url = props.url || '';

    container.innerHTML = `
      <div class="widget-header">
        <div class="widget-title-group">
          <span class="widget-icon">${Icons.get('layers', 18)}</span>
          <span class="widget-title">${this.escapeHTML(widget.title)}</span>
        </div>
      </div>
      ${url ? `<iframe src="${url}" style="width: 100%; height: 260px; border: none; border-radius: var(--radius-md);" loading="lazy"></iframe>` : '<div class="text-dim">Configure IFrame URL in settings</div>'}
    `;
  },

  renderGeneric(container, widget) {
    container.innerHTML = `
      <div class="widget-header">
        <span class="widget-title">${this.escapeHTML(widget.title)}</span>
      </div>
      <div class="text-dim">Custom widget (${this.escapeHTML(widget.type)})</div>
    `;
  },

  escapeHTML(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }
};
