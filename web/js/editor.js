/**
 * Homebase WYSIWYG Editor Module
 * Manages in-place dashboard layout editing, live widget resizing,
 * tab creation/reordering, and configuration persistence.
 */

const Editor = {
  isEditing: false,
  isDirty: false,
  activeWidgetEditingId: null,

  init() {
    this.setupListeners();
    this.buildCatalog();
  },

  setupListeners() {
    // Edit mode toggle
    const toggleBtn = document.getElementById('btn-toggle-edit');
    if (toggleBtn) {
      toggleBtn.onclick = () => this.toggleEditMode();
    }

    // Save changes button
    const saveBtn = document.getElementById('btn-save-config');
    if (saveBtn) {
      saveBtn.onclick = () => this.saveChanges();
    }

    // Modal close buttons
    document.querySelectorAll('[data-close-modal]').forEach(btn => {
      btn.onclick = () => {
        const modalId = btn.getAttribute('data-close-modal');
        this.closeModal(modalId);
      };
    });

    // Save Widget modal button
    const saveWidgetBtn = document.getElementById('btn-save-widget');
    if (saveWidgetBtn) {
      saveWidgetBtn.onclick = () => this.saveWidgetForm();
    }

    // Add Tab button
    const addTabBtn = document.getElementById('btn-add-tab');
    if (addTabBtn) {
      addTabBtn.onclick = () => this.openTabModal(null);
    }

    // Save Tab button
    const saveTabBtn = document.getElementById('btn-save-tab');
    if (saveTabBtn) {
      saveTabBtn.onclick = () => this.saveTabForm();
    }

    // Delete Tab button
    const deleteTabBtn = document.getElementById('btn-delete-tab');
    if (deleteTabBtn) {
      deleteTabBtn.onclick = () => this.deleteCurrentTab();
    }
  },

  toggleEditMode() {
    this.isEditing = !this.isEditing;
    document.body.classList.toggle('editing-active', this.isEditing);

    const toggleBtn = document.getElementById('btn-toggle-edit');
    const label = document.getElementById('edit-toggle-label');
    const saveBtn = document.getElementById('btn-save-config');

    if (this.isEditing) {
      toggleBtn.classList.add('active');
      label.textContent = 'Exit Edit Mode';
      if (this.isDirty) {
        saveBtn.style.display = 'inline-flex';
      }
      App.showToast('Edit Mode enabled: click actions on cards to resize or move.', 'info');
    } else {
      toggleBtn.classList.remove('active');
      label.textContent = 'Edit Dashboard';
      saveBtn.style.display = 'none';

      // Auto-save on exit if dirty
      if (this.isDirty) {
        this.saveChanges();
      }
    }

    App.renderCurrentTab();
  },

  markDirty() {
    this.isDirty = true;
    const saveBtn = document.getElementById('btn-save-config');
    if (saveBtn && this.isEditing) {
      saveBtn.style.display = 'inline-flex';
    }
  },

  async saveChanges() {
    try {
      await API.saveConfig(App.config);
      this.isDirty = false;
      const saveBtn = document.getElementById('btn-save-config');
      if (saveBtn) saveBtn.style.display = 'none';
      App.showToast('Dashboard configuration saved successfully', 'success');
    } catch (err) {
      App.showToast(`Failed to save config: ${err.message}`, 'error');
    }
  },

  // ==================== WIDGET MANAGEMENT ====================

  moveWidget(widgetId, delta) {
    const tab = App.getCurrentTab();
    if (!tab) return;

    const idx = tab.widgets.findIndex(w => w.id === widgetId);
    if (idx < 0) return;

    const targetIdx = idx + delta;
    if (targetIdx < 0 || targetIdx >= tab.widgets.length) return;

    const [moved] = tab.widgets.splice(idx, 1);
    tab.widgets.splice(targetIdx, 0, moved);

    this.markDirty();
    App.renderCurrentTab();
  },

  resizeWidget(widgetId, newColSpan) {
    const tab = App.getCurrentTab();
    if (!tab) return;

    const w = tab.widgets.find(x => x.id === widgetId);
    if (!w) return;

    w.colSpan = Math.max(1, Math.min(4, newColSpan));
    this.markDirty();
    App.renderCurrentTab();
  },

  deleteWidget(widgetId) {
    const tab = App.getCurrentTab();
    if (!tab) return;

    const idx = tab.widgets.findIndex(w => w.id === widgetId);
    if (idx >= 0) {
      tab.widgets.splice(idx, 1);
      this.markDirty();
      App.renderCurrentTab();
      App.showToast('Widget deleted', 'info');
    }
  },

  openAddWidgetModal() {
    this.openModal('modal-add-widget');
  },

  buildCatalog() {
    const catalog = document.getElementById('widget-catalog');
    if (!catalog) return;

    const templates = [
      {
        type: 'service',
        title: 'Service Bookmark',
        desc: 'Quick link with health status, ping latency, custom icon and description',
        icon: 'server',
        defaultProps: { url: 'https://', description: '', icon: 'server', tag: 'Web', pingUrl: '' },
        defaultCol: 1,
        defaultRow: 1
      },
      {
        type: 'proxmox',
        title: 'Proxmox VE Cluster',
        desc: 'Live CPU, RAM, disk, and VM/LXC metrics via Proxmox API token',
        icon: 'server',
        defaultProps: { url: 'https://pve.local:8006', apiToken: '', node: 'pve', insecureSkipVerify: true },
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'truenas',
        title: 'TrueNAS Pool & Health',
        desc: 'Storage pool capacity gauge, pool health status, and active alert count',
        icon: 'hard-drive',
        defaultProps: { url: 'https://truenas.local', apiKey: '', pool: 'tank', insecureSkipVerify: true },
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'system',
        title: 'Host Resources',
        desc: 'Host CPU cores, memory allocation, goroutines, and application uptime',
        icon: 'cpu',
        defaultProps: {},
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'docker',
        title: 'Docker Containers',
        desc: 'Running & stopped container overview from local Docker socket',
        icon: 'box',
        defaultProps: { socket: '/var/run/docker.sock' },
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'weather',
        title: 'Weather Forecast',
        desc: 'Real-time temperature and condition via Open-Meteo',
        icon: 'cloud',
        defaultProps: { city: 'New York', latitude: 40.7128, longitude: -74.0060, units: 'celsius' },
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'clock',
        title: 'Clock & Date',
        desc: 'Digital clock with live seconds, weekday, and date',
        icon: 'clock',
        defaultProps: { format24: true },
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'search',
        title: 'Web Search Bar',
        desc: 'Multi-engine search box supporting DuckDuckGo, Google, Bing, Reddit, SearXNG',
        icon: 'search',
        defaultProps: { provider: 'duckduckgo' },
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'notes',
        title: 'Notes & Documentation',
        desc: 'Homelab scratchpad, cheat-sheet, or server IP addresses',
        icon: 'file-text',
        defaultProps: { content: 'Notes...' },
        defaultCol: 2,
        defaultRow: 1
      },
      {
        type: 'iframe',
        title: 'Web Embed / IFrame',
        desc: 'Embed external web pages like Grafana, Home Assistant, or Pi-hole',
        icon: 'layers',
        defaultProps: { url: 'https://' },
        defaultCol: 4,
        defaultRow: 2
      }
    ];

    catalog.innerHTML = templates.map(t => `
      <div class="catalog-item" onclick="Editor.createWidgetFromTemplate('${t.type}', '${t.title}', ${t.defaultCol}, ${t.defaultRow})">
        <div class="catalog-item-header">
          <div class="catalog-item-icon">${Icons.get(t.icon, 18)}</div>
          <div class="catalog-item-title">${t.title}</div>
        </div>
        <div class="catalog-item-desc">${t.desc}</div>
      </div>
    `).join('');
  },

  createWidgetFromTemplate(type, title, defaultCol = 2, defaultRow = 1) {
    const tab = App.getCurrentTab();
    if (!tab) return;

    const newWidget = {
      id: `w-${Date.now()}-${Math.random().toString(36).substr(2, 4)}`,
      title: title,
      type: type,
      colSpan: defaultCol,
      rowSpan: defaultRow,
      order: tab.widgets.length + 1,
      properties: {}
    };

    // Pre-populate sensible defaults
    if (type === 'service') {
      newWidget.properties = { url: 'https://', description: 'Application link', icon: 'server', tag: 'Web' };
    } else if (type === 'proxmox') {
      newWidget.properties = { url: 'https://pve.local:8006', apiToken: '', node: 'pve', insecureSkipVerify: true };
    } else if (type === 'truenas') {
      newWidget.properties = { url: 'https://truenas.local', apiKey: '', pool: 'tank', insecureSkipVerify: true };
    } else if (type === 'weather') {
      newWidget.properties = { city: 'New York', latitude: 40.7128, longitude: -74.0060, units: 'celsius' };
    } else if (type === 'docker') {
      newWidget.properties = { socket: '/var/run/docker.sock' };
    }

    tab.widgets.push(newWidget);
    this.closeModal('modal-add-widget');
    this.markDirty();
    App.renderCurrentTab();

    // Immediately open configuration modal so user can fill in URLs or API tokens
    setTimeout(() => {
      this.openWidgetModal(newWidget.id);
    }, 150);
  },

  openWidgetModal(widgetId) {
    const tab = App.getCurrentTab();
    if (!tab) return;

    const w = tab.widgets.find(x => x.id === widgetId);
    if (!w) return;

    this.activeWidgetEditingId = widgetId;
    document.getElementById('widget-id').value = w.id;
    document.getElementById('widget-type').value = w.type;
    document.getElementById('widget-title').value = w.title || '';
    document.getElementById('widget-colspan').value = String(w.colSpan || 2);
    document.getElementById('widget-rowspan').value = String(w.rowSpan || 1);

    const propContainer = document.getElementById('widget-properties-fields');
    propContainer.innerHTML = '';

    const props = w.properties || {};

    if (w.type === 'service') {
      propContainer.innerHTML = `
        <div class="form-group col-span-2">
          <label>Destination URL</label>
          <input type="text" id="prop-url" class="form-input" value="${this.escapeAttr(props.url || '')}" placeholder="https://app.local:8080">
        </div>
        <div class="form-group col-span-2">
          <label>Description</label>
          <input type="text" id="prop-description" class="form-input" value="${this.escapeAttr(props.description || '')}" placeholder="Primary management console">
        </div>
        <div class="form-group">
          <label>Icon</label>
          <select id="prop-icon" class="form-select">
            ${this.buildIconOptions(props.icon || 'server')}
          </select>
        </div>
        <div class="form-group">
          <label>Category Tag</label>
          <input type="text" id="prop-tag" class="form-input" value="${this.escapeAttr(props.tag || '')}" placeholder="e.g. Media, Network, Storage">
        </div>
        <div class="form-group col-span-2">
          <label>Health Check / Ping URL (optional)</label>
          <input type="text" id="prop-pingUrl" class="form-input" value="${this.escapeAttr(props.pingUrl || '')}" placeholder="https://app.local:8080 (defaults to Destination URL)">
        </div>
      `;
    } else if (w.type === 'proxmox') {
      propContainer.innerHTML = `
        <div class="form-group col-span-2">
          <label>Proxmox VE URL</label>
          <input type="text" id="prop-pve-url" class="form-input" value="${this.escapeAttr(props.url || '')}" placeholder="https://pve.local:8006">
        </div>
        <div class="form-group col-span-2">
          <label>Proxmox API Token</label>
          <input type="text" id="prop-pve-token" class="form-input" value="${this.escapeAttr(props.apiToken || '')}" placeholder="root@pam!homebase=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx">
          <small class="form-help">Format: USER@REALM!TOKENID=UUID. Safe backend proxying handles requests.</small>
        </div>
        <div class="form-group">
          <label>Node Name</label>
          <input type="text" id="prop-pve-node" class="form-input" value="${this.escapeAttr(props.node || 'pve')}" placeholder="pve">
        </div>
        <div class="form-group">
          <label>Allow Self-Signed TLS</label>
          <div class="checkbox-row" style="margin-top: 0.5rem;">
            <input type="checkbox" id="prop-pve-insecure" class="checkbox-input"${props.insecureSkipVerify !== false ? ' checked' : ''}>
            <label for="prop-pve-insecure">Ignore SSL Certificate warnings</label>
          </div>
        </div>
      `;
    } else if (w.type === 'truenas') {
      propContainer.innerHTML = `
        <div class="form-group col-span-2">
          <label>TrueNAS SCALE / CORE URL</label>
          <input type="text" id="prop-tn-url" class="form-input" value="${this.escapeAttr(props.url || '')}" placeholder="https://truenas.local">
        </div>
        <div class="form-group col-span-2">
          <label>TrueNAS API Key</label>
          <input type="text" id="prop-tn-key" class="form-input" value="${this.escapeAttr(props.apiKey || '')}" placeholder="API Key generated in TrueNAS UI">
        </div>
        <div class="form-group">
          <label>Primary Storage Pool Name</label>
          <input type="text" id="prop-tn-pool" class="form-input" value="${this.escapeAttr(props.pool || 'tank')}" placeholder="tank">
        </div>
        <div class="form-group">
          <label>Allow Self-Signed TLS</label>
          <div class="checkbox-row" style="margin-top: 0.5rem;">
            <input type="checkbox" id="prop-tn-insecure" class="checkbox-input"${props.insecureSkipVerify !== false ? ' checked' : ''}>
            <label for="prop-tn-insecure">Ignore SSL Certificate warnings</label>
          </div>
        </div>
      `;
    } else if (w.type === 'weather') {
      propContainer.innerHTML = `
        <div class="form-group col-span-2">
          <label>City Label</label>
          <input type="text" id="prop-weather-city" class="form-input" value="${this.escapeAttr(props.city || '')}" placeholder="New York">
        </div>
        <div class="form-group">
          <label>Latitude</label>
          <input type="number" step="0.0001" id="prop-weather-lat" class="form-input" value="${props.latitude || 40.7128}">
        </div>
        <div class="form-group">
          <label>Longitude</label>
          <input type="number" step="0.0001" id="prop-weather-lon" class="form-input" value="${props.longitude || -74.0060}">
        </div>
        <div class="form-group">
          <label>Units</label>
          <select id="prop-weather-units" class="form-select">
            <option value="celsius"${props.units !== 'fahrenheit' ? ' selected' : ''}>Celsius (°C)</option>
            <option value="fahrenheit"${props.units === 'fahrenheit' ? ' selected' : ''}>Fahrenheit (°F)</option>
          </select>
        </div>
      `;
    } else if (w.type === 'docker') {
      propContainer.innerHTML = `
        <div class="form-group col-span-2">
          <label>Docker Socket Path or TCP URL</label>
          <input type="text" id="prop-docker-socket" class="form-input" value="${this.escapeAttr(props.socket || '/var/run/docker.sock')}" placeholder="/var/run/docker.sock or http://127.0.0.1:2375">
        </div>
      `;
    } else if (w.type === 'notes') {
      propContainer.innerHTML = `
        <div class="form-group col-span-2">
          <label>Note Content</label>
          <textarea id="prop-notes-content" class="form-textarea" rows="6">${this.escapeHTML(props.content || '')}</textarea>
        </div>
      `;
    } else if (w.type === 'iframe') {
      propContainer.innerHTML = `
        <div class="form-group col-span-2">
          <label>IFrame Target URL</label>
          <input type="text" id="prop-iframe-url" class="form-input" value="${this.escapeAttr(props.url || '')}" placeholder="https://grafana.local:3000">
        </div>
      `;
    }

    this.openModal('modal-edit-widget');
  },

  saveWidgetForm() {
    const tab = App.getCurrentTab();
    if (!tab || !this.activeWidgetEditingId) return;

    const w = tab.widgets.find(x => x.id === this.activeWidgetEditingId);
    if (!w) return;

    w.title = document.getElementById('widget-title').value.trim() || w.title;
    w.colSpan = parseInt(document.getElementById('widget-colspan').value, 10);
    w.rowSpan = parseInt(document.getElementById('widget-rowspan').value, 10);

    w.properties = w.properties || {};

    if (w.type === 'service') {
      w.properties.url = (document.getElementById('prop-url')?.value || '').trim();
      w.properties.description = (document.getElementById('prop-description')?.value || '').trim();
      w.properties.icon = document.getElementById('prop-icon')?.value || 'server';
      w.properties.tag = (document.getElementById('prop-tag')?.value || '').trim();
      w.properties.pingUrl = (document.getElementById('prop-pingUrl')?.value || '').trim();
    } else if (w.type === 'proxmox') {
      w.properties.url = (document.getElementById('prop-pve-url')?.value || '').trim();
      w.properties.apiToken = (document.getElementById('prop-pve-token')?.value || '').trim();
      w.properties.node = (document.getElementById('prop-pve-node')?.value || 'pve').trim();
      w.properties.insecureSkipVerify = document.getElementById('prop-pve-insecure')?.checked !== false;
    } else if (w.type === 'truenas') {
      w.properties.url = (document.getElementById('prop-tn-url')?.value || '').trim();
      w.properties.apiKey = (document.getElementById('prop-tn-key')?.value || '').trim();
      w.properties.pool = (document.getElementById('prop-tn-pool')?.value || 'tank').trim();
      w.properties.insecureSkipVerify = document.getElementById('prop-tn-insecure')?.checked !== false;
    } else if (w.type === 'weather') {
      w.properties.city = (document.getElementById('prop-weather-city')?.value || '').trim();
      w.properties.latitude = parseFloat(document.getElementById('prop-weather-lat')?.value) || 0;
      w.properties.longitude = parseFloat(document.getElementById('prop-weather-lon')?.value) || 0;
      w.properties.units = document.getElementById('prop-weather-units')?.value || 'celsius';
    } else if (w.type === 'docker') {
      w.properties.socket = (document.getElementById('prop-docker-socket')?.value || '').trim();
    } else if (w.type === 'notes') {
      w.properties.content = document.getElementById('prop-notes-content')?.value || '';
    } else if (w.type === 'iframe') {
      w.properties.url = (document.getElementById('prop-iframe-url')?.value || '').trim();
    }

    this.closeModal('modal-edit-widget');
    this.markDirty();
    App.renderCurrentTab();
    App.showToast(`Updated "${w.title}"`, 'success');
  },

  // ==================== TAB MANAGEMENT ====================

  openTabModal(tabId = null) {
    const modalTitle = document.getElementById('tab-modal-title');
    const deleteBtn = document.getElementById('btn-delete-tab');
    const idInput = document.getElementById('tab-id');
    const nameInput = document.getElementById('tab-name');
    const iconSelect = document.getElementById('tab-icon');

    if (tabId) {
      const tab = App.config.tabs.find(t => t.id === tabId);
      if (!tab) return;
      modalTitle.textContent = 'Edit Tab';
      idInput.value = tab.id;
      nameInput.value = tab.name;
      iconSelect.value = tab.icon || 'layout-dashboard';
      deleteBtn.style.display = App.config.tabs.length > 1 ? 'inline-block' : 'none';
    } else {
      modalTitle.textContent = 'New Tab';
      idInput.value = '';
      nameInput.value = '';
      iconSelect.value = 'grid';
      deleteBtn.style.display = 'none';
    }

    this.openModal('modal-tab-editor');
  },

  saveTabForm() {
    const id = document.getElementById('tab-id').value;
    const name = document.getElementById('tab-name').value.trim();
    const icon = document.getElementById('tab-icon').value;

    if (!name) {
      App.showToast('Please enter a tab name', 'error');
      return;
    }

    if (id) {
      const tab = App.config.tabs.find(t => t.id === id);
      if (tab) {
        tab.name = name;
        tab.icon = icon;
      }
    } else {
      const newTabId = 'tab-' + Date.now();
      App.config.tabs.push({
        id: newTabId,
        name: name,
        icon: icon,
        widgets: []
      });
      App.activeTabId = newTabId;
    }

    this.closeModal('modal-tab-editor');
    this.markDirty();
    App.renderTabs();
    App.renderCurrentTab();
    App.showToast(`Tab saved`, 'success');
  },

  deleteCurrentTab() {
    const id = document.getElementById('tab-id').value;
    if (!id || App.config.tabs.length <= 1) return;

    const idx = App.config.tabs.findIndex(t => t.id === id);
    if (idx >= 0) {
      App.config.tabs.splice(idx, 1);
      App.activeTabId = App.config.tabs[0].id;
      this.closeModal('modal-tab-editor');
      this.markDirty();
      App.renderTabs();
      App.renderCurrentTab();
      App.showToast('Tab removed', 'info');
    }
  },

  // ==================== UTILS ====================

  openModal(id) {
    const el = document.getElementById(id);
    if (el) el.classList.add('open');
  },

  closeModal(id) {
    const el = document.getElementById(id);
    if (el) el.classList.remove('open');
  },

  buildIconOptions(selected) {
    const iconList = [
      'server', 'hard-drive', 'cpu', 'shield', 'box', 'film', 'home',
      'activity', 'check-circle', 'clock', 'cloud', 'terminal', 'file-text',
      'grid', 'layers', 'code', 'layout-dashboard'
    ];

    return iconList.map(i => `<option value="${i}"${i === selected ? ' selected' : ''}>${i}</option>`).join('');
  },

  escapeHTML(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  },

  escapeAttr(str) {
    if (!str) return '';
    return String(str)
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }
};
