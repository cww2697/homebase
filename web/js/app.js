/**
 * Homebase Application Controller
 * Bootstraps state, tab rendering, live metric polling,
 * settings, and Git synchronization.
 */

const App = {
  config: null,
  activeTabId: null,
  refreshTimer: null,

  async init() {
    this.setupBrandIcon();
    this.setupHeaderClock();
    this.setupModals();
    this.setupExportImport();
    Editor.init();

    await this.loadConfig();
    this.startPeriodicRefresh();
  },

  setupBrandIcon() {
    const iconEl = document.getElementById('brand-icon');
    if (iconEl) {
      iconEl.innerHTML = Icons.get('layout-dashboard', 22);
    }
    Icons.renderAll(document);
  },

  setupHeaderClock() {
    const updateTime = () => {
      const now = new Date();
      const timeEl = document.getElementById('header-clock-time');
      const dateEl = document.getElementById('header-clock-date');
      if (timeEl) {
        timeEl.textContent = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
      }
      if (dateEl) {
        dateEl.textContent = now.toLocaleDateString([], { month: 'short', day: 'numeric' });
      }
    };
    updateTime();
    setInterval(updateTime, 1000);
  },

  async loadConfig() {
    try {
      this.config = await API.getConfig();
      this.applySettings(this.config.settings);
      this.updateGitIndicator(this.config.git);

      if (!this.config.tabs || this.config.tabs.length === 0) {
        this.config.tabs = [{ id: 'home', name: 'Home', icon: 'layout-dashboard', widgets: [] }];
      }

      if (!this.activeTabId || !this.config.tabs.some(t => t.id === this.activeTabId)) {
        this.activeTabId = this.config.tabs[0].id;
      }

      this.renderTabs();
      this.renderCurrentTab();
    } catch (err) {
      console.error('Failed to load initial configuration:', err);
      this.showToast('Failed to load configuration from server', 'error');
    }
  },

  applySettings(settings = {}) {
    if (!settings) return;

    // Title
    const titleEl = document.getElementById('dashboard-title');
    if (titleEl) {
      titleEl.textContent = settings.title || 'Homebase';
      document.title = `${settings.title || 'Homebase'} | Homelab Dashboard`;
    }

    // Theme
    const theme = settings.theme || 'dark';
    document.body.className = `theme-${theme}`;

    // Columns
    const cols = settings.columns || 4;
    document.documentElement.style.setProperty('--grid-columns', cols);

    // Background Image & Blur
    if (settings.backgroundUrl) {
      document.body.style.backgroundImage = `url('${settings.backgroundUrl}')`;
      document.body.classList.add('has-bg-image');
    } else {
      document.body.style.backgroundImage = '';
      document.body.classList.remove('has-bg-image');
    }

    if (settings.backgroundBlur) {
      document.body.classList.add('blur-active');
    } else {
      document.body.classList.remove('blur-active');
    }
  },

  updateGitIndicator(gitConfig = {}) {
    const indicator = document.getElementById('git-indicator');
    if (indicator) {
      if (gitConfig.enabled) {
        indicator.classList.add('active');
      } else {
        indicator.classList.remove('active');
      }
    }
  },

  getCurrentTab() {
    if (!this.config || !this.config.tabs) return null;
    return this.config.tabs.find(t => t.id === this.activeTabId) || this.config.tabs[0];
  },

  renderTabs() {
    const container = document.getElementById('tabs-container');
    if (!container || !this.config || !this.config.tabs) return;

    container.innerHTML = this.config.tabs.map(tab => {
      const isActive = tab.id === this.activeTabId;
      return `
        <div class="tab-btn ${isActive ? 'active' : ''}" onclick="App.selectTab('${tab.id}')">
          <span class="icon-slot">${Icons.get(tab.icon || 'layout-dashboard', 16)}</span>
          <span>${this.escapeHTML(tab.name)}</span>
          <span class="tab-edit-controls">
            <span class="tab-action-icon" title="Edit Tab" onclick="event.stopPropagation(); Editor.openTabModal('${tab.id}')">
              ${Icons.get('edit-3', 12)}
            </span>
          </span>
        </div>
      `;
    }).join('');
  },

  selectTab(tabId) {
    this.activeTabId = tabId;
    this.renderTabs();
    this.renderCurrentTab();
  },

  renderCurrentTab() {
    const grid = document.getElementById('widgets-grid');
    if (!grid) return;

    grid.innerHTML = '';
    const tab = this.getCurrentTab();
    if (!tab) return;

    // Render widgets
    tab.widgets.forEach(w => {
      const card = Widgets.render(w, Editor.isEditing);
      grid.appendChild(card);
    });

    // Render "+ Add Widget" card (visible in edit mode)
    const addCard = document.createElement('div');
    addCard.className = 'add-widget-placeholder col-span-2';
    addCard.innerHTML = `
      <div class="add-widget-icon">${Icons.get('plus', 22)}</div>
      <span class="font-medium">Add Widget</span>
    `;
    addCard.onclick = () => Editor.openAddWidgetModal();
    grid.appendChild(addCard);

    Icons.renderAll(grid);
  },

  startPeriodicRefresh() {
    if (this.refreshTimer) clearInterval(this.refreshTimer);

    const intervalSec = (this.config?.settings?.refreshInterval) || 30;
    this.refreshTimer = setInterval(() => {
      this.refreshMetrics();
    }, intervalSec * 1000);
  },

  refreshMetrics() {
    const tab = this.getCurrentTab();
    if (!tab) return;

    tab.widgets.forEach(w => {
      if (w.type === 'proxmox') Widgets.refreshProxmox(w);
      if (w.type === 'truenas') Widgets.refreshTrueNAS(w);
      if (w.type === 'system') Widgets.refreshSystem(w);
      if (w.type === 'docker') Widgets.refreshDocker(w);
      if (w.type === 'weather') Widgets.refreshWeather(w);
      if (w.type === 'service') {
        const pingTarget = w.properties?.pingUrl || w.properties?.url;
        if (pingTarget) Widgets.checkServiceHealth(w.id, pingTarget);
      }
    });
  },

  setupModals() {
    // Settings modal trigger
    const settingsBtn = document.getElementById('btn-open-settings');
    if (settingsBtn) {
      settingsBtn.onclick = () => this.openSettingsModal();
    }

    // Settings save button
    const saveSettingsBtn = document.getElementById('btn-save-settings');
    if (saveSettingsBtn) {
      saveSettingsBtn.onclick = () => this.saveSettingsModal();
    }

    // Reset defaults button
    const resetBtn = document.getElementById('btn-reset-defaults');
    if (resetBtn) {
      resetBtn.onclick = () => this.resetDefaults();
    }

    // Git modal trigger
    const gitBtn = document.getElementById('btn-open-git');
    if (gitBtn) {
      gitBtn.onclick = () => this.openGitModal();
    }

    // Git save button
    const saveGitBtn = document.getElementById('btn-save-git-config');
    if (saveGitBtn) {
      saveGitBtn.onclick = () => this.saveGitModal();
    }

    // Git Pull & Push buttons
    const pullBtn = document.getElementById('btn-git-pull');
    if (pullBtn) {
      pullBtn.onclick = () => this.triggerGitSync('pull');
    }

    const pushBtn = document.getElementById('btn-git-push');
    if (pushBtn) {
      pushBtn.onclick = () => this.triggerGitSync('push');
    }
  },

  openSettingsModal() {
    const s = this.config.settings || {};
    document.getElementById('setting-title').value = s.title || 'Homebase';
    document.getElementById('setting-theme').value = s.theme || 'dark';
    document.getElementById('setting-columns').value = String(s.columns || 4);
    document.getElementById('setting-refresh').value = s.refreshInterval || 30;
    document.getElementById('setting-bg-blur').checked = s.backgroundBlur !== false;
    document.getElementById('setting-bg-url').value = s.backgroundUrl || '';

    Editor.openModal('modal-settings');
  },

  async saveSettingsModal() {
    this.config.settings = {
      title: document.getElementById('setting-title').value.trim() || 'Homebase',
      theme: document.getElementById('setting-theme').value,
      columns: parseInt(document.getElementById('setting-columns').value, 10) || 4,
      refreshInterval: parseInt(document.getElementById('setting-refresh').value, 10) || 30,
      backgroundBlur: document.getElementById('setting-bg-blur').checked,
      backgroundUrl: document.getElementById('setting-bg-url').value.trim()
    };

    this.applySettings(this.config.settings);
    this.startPeriodicRefresh();
    Editor.closeModal('modal-settings');
    Editor.markDirty();
    await Editor.saveChanges();
  },

  async resetDefaults() {
    if (!confirm('Are you sure you want to reset the dashboard to defaults? All custom tabs and widgets will be restored.')) {
      return;
    }

    try {
      const res = await API.resetConfig();
      this.config = res.config;
      this.activeTabId = this.config.tabs[0]?.id;
      this.applySettings(this.config.settings);
      this.renderTabs();
      this.renderCurrentTab();
      Editor.closeModal('modal-settings');
      this.showToast('Restored default dashboard configuration', 'success');
    } catch (err) {
      this.showToast(`Reset failed: ${err.message}`, 'error');
    }
  },

  openGitModal() {
    const g = this.config.git || {};
    document.getElementById('git-enabled').checked = !!g.enabled;
    document.getElementById('git-repo-url').value = g.repoUrl || '';
    document.getElementById('git-branch').value = g.branch || 'main';
    document.getElementById('git-token').value = g.token || '';
    document.getElementById('git-auto-save').checked = !!g.autoSyncOnSave;
    document.getElementById('git-auto-startup').checked = !!g.autoSyncOnStartup;

    const statusText = document.getElementById('git-status-text');
    if (statusText) {
      statusText.textContent = g.lastSyncStatus || 'Not configured';
      statusText.style.color = g.lastSyncStatus === 'success' ? 'var(--accent)' : (g.lastSyncStatus === 'error' ? 'var(--danger)' : 'var(--text-muted)');
    }

    Editor.openModal('modal-git');
  },

  async saveGitModal() {
    this.config.git = {
      enabled: document.getElementById('git-enabled').checked,
      repoUrl: document.getElementById('git-repo-url').value.trim(),
      branch: document.getElementById('git-branch').value.trim() || 'main',
      token: document.getElementById('git-token').value.trim(),
      autoSyncOnSave: document.getElementById('git-auto-save').checked,
      autoSyncOnStartup: document.getElementById('git-auto-startup').checked,
      lastSyncStatus: this.config.git?.lastSyncStatus || 'unconfigured'
    };

    this.updateGitIndicator(this.config.git);
    Editor.closeModal('modal-git');
    Editor.markDirty();
    await Editor.saveChanges();
  },

  async triggerGitSync(action) {
    const outputBox = document.getElementById('git-output-box');
    const outputText = document.getElementById('git-output-text');
    if (outputBox) outputBox.style.display = 'block';
    if (outputText) outputText.textContent = `Running Git ${action.toUpperCase()}...`;

    try {
      const res = await API.syncGit(action, `Homebase ${action} sync`);
      if (outputText) {
        outputText.textContent = `[${res.action.toUpperCase()} Result]\nSuccess: ${res.success}\nMessage: ${res.message}`;
      }

      if (res.success) {
        this.showToast(`Git ${action} succeeded`, 'success');
        if (action === 'pull') {
          // Reload updated config
          await this.loadConfig();
        }
      } else {
        this.showToast(`Git ${action} failed: ${res.message}`, 'error');
      }
    } catch (err) {
      if (outputText) outputText.textContent = `Error: ${err.message}`;
      this.showToast(`Git sync failed: ${err.message}`, 'error');
    }
  },

  setupExportImport() {
    // Export JSON
    const exportBtn = document.getElementById('btn-export-config');
    if (exportBtn) {
      exportBtn.onclick = () => {
        const jsonStr = JSON.stringify(this.config, null, 2);
        const blob = new Blob([jsonStr], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `homebase-config-${new Date().toISOString().slice(0, 10)}.json`;
        a.click();
        URL.revokeObjectURL(url);
        this.showToast('Configuration exported as JSON', 'info');
      };
    }

    // Import JSON
    const importBtn = document.getElementById('btn-import-config');
    const importInput = document.getElementById('file-import-input');
    if (importBtn && importInput) {
      importBtn.onclick = () => importInput.click();
      importInput.onchange = async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        try {
          const text = await file.text();
          const parsed = JSON.parse(text);
          if (!parsed.tabs || !Array.isArray(parsed.tabs)) {
            throw new Error('Missing tabs array in JSON');
          }

          await API.saveConfig(parsed);
          await this.loadConfig();
          this.showToast('Configuration imported successfully', 'success');
        } catch (err) {
          this.showToast(`Import failed: ${err.message}`, 'error');
        }
        importInput.value = '';
      };
    }
  },

  showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    let iconName = 'activity';
    if (type === 'success') iconName = 'check-circle';
    if (type === 'error') iconName = 'shield';

    toast.innerHTML = `
      <span class="icon-slot">${Icons.get(iconName, 16)}</span>
      <span>${this.escapeHTML(message)}</span>
    `;

    container.appendChild(toast);
    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transition = 'opacity 0.3s ease';
      setTimeout(() => toast.remove(), 300);
    }, 4000);
  },

  escapeHTML(str) {
    if (!str) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }
};

// Start application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  App.init();
});
