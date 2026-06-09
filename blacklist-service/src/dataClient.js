// Adapter for data service.
// Domains passed in are assumed already normalised/validated by the caller.

export class DataClient {
    /**
     * @param {object} options
     * @param {string} options.baseUrl   e.g. http://data-service:9090
     * @param {string} options.apiKey    shared Bearer secret
     * @param {number} [options.timeoutMs]
     */
    constructor({ baseUrl, apiKey, timeoutMs = 5000 }) {
        this.baseUrl = baseUrl.replace(/\/$/, '');
        this.apiKey = apiKey;
        this.timeoutMs = timeoutMs;
    }

    async #request(method, path, body) {
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), this.timeoutMs);
        try {
            const res = await fetch(`${this.baseUrl}${path}`, {
                method,
                headers: {
                    Authorization: `Bearer ${this.apiKey}`,
                    ...(body ? { 'Content-Type': 'application/json' } : {}),
                },
                body: body ? JSON.stringify(body) : undefined,
                signal: controller.signal,
            });
            return res;
        } finally {
            clearTimeout(timer);
        }
    }

    async list() {
        const res = await this.#request('GET', '/blacklist');
        if (!res.ok) throw new Error(`data-service list failed: ${res.status}`);
        const body = await res.json();
        return Array.isArray(body.domains) ? body.domains : [];
    }

    async add(domain) {
        const res = await this.#request('POST', '/blacklist', { domain });
        if (res.status === 201) return true; // newly added
        if (res.status === 200) return false; // already present
        throw new Error(`data-service add failed: ${res.status}`);
    }

    async remove(domain) {
        const res = await this.#request('DELETE', `/blacklist/${encodeURIComponent(domain)}`);
        if (res.status === 200) return true;
        if (res.status === 404) return false;
        throw new Error(`data-service remove failed: ${res.status}`);
    }

    async isBlacklisted(domain) {
        const res = await this.#request(
            'GET',
            `/blacklist/check?domain=${encodeURIComponent(domain)}`,
        );
        if (!res.ok) throw new Error(`data-service check failed: ${res.status}`);
        const body = await res.json();
        return Boolean(body.blacklisted);
    }
}
