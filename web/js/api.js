/**
 * Homebase API Client
 * Wraps REST calls to backend server endpoints
 */

const API = {
  baseURL: '',

  async request(endpoint, options = {}) {
    try {
      const url = this.baseURL + endpoint;
      const res = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {})
        },
        ...options
      });

      if (!res.ok) {
        const errorText = await res.text();
        let message = `HTTP ${res.status}`;
        try {
          const jsonErr = JSON.parse(errorText);
          message = jsonErr.error || message;
        } catch (_) {}
        throw new Error(message);
      }

      return await res.json();
    } catch (err) {
      console.warn(`API call failed for ${endpoint}:`, err);
      throw err;
    }
  },

  async getConfig() {
    return await this.request('/api/config');
  },

  async saveConfig(cfg) {
    return await this.request('/api/config', {
      method: 'POST',
      body: JSON.stringify(cfg)
    });
  },

  async resetConfig() {
    return await this.request('/api/config/reset', {
      method: 'POST'
    });
  },

  async syncGit(action = 'pull', message = '') {
    return await this.request('/api/git/sync', {
      method: 'POST',
      body: JSON.stringify({ action, message })
    });
  },

  async queryProxmox(query) {
    return await this.request('/api/proxy/proxmox', {
      method: 'POST',
      body: JSON.stringify(query)
    });
  },

  async queryTrueNAS(query) {
    return await this.request('/api/proxy/truenas', {
      method: 'POST',
      body: JSON.stringify(query)
    });
  },

  async checkPing(url) {
    return await this.request('/api/proxy/ping', {
      method: 'POST',
      body: JSON.stringify({ url })
    });
  },

  async getSystemStats() {
    return await this.request('/api/system/stats');
  },

  async queryDocker(socketPath) {
    return await this.request('/api/proxy/docker', {
      method: 'POST',
      body: JSON.stringify({ socketPath })
    });
  },

  async getWeather(lat, lon, city, units) {
    const params = new URLSearchParams();
    if (lat) params.set('lat', lat);
    if (lon) params.set('lon', lon);
    if (city) params.set('city', city);
    if (units) params.set('units', units);
    return await this.request(`/api/weather?${params.toString()}`);
  }
};
