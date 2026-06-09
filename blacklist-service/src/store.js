// In-memory blacklist store.
export class BlacklistStore {
  /** @param {string[]} [seed] initial domains (assumed already normalised) */
  constructor(seed = []) {
    /** @type {Set<string>} */
    this.domains = new Set(seed);
  }

  /** @returns {string[]} sorted list of blacklisted domains */
  list() {
    return [...this.domains].sort();
  }

  /**
   * @param {string} domain canonical domain
   * @returns {boolean} true if newly added, false if it already existed
   */
  add(domain) {
    if (this.domains.has(domain)) return false;
    this.domains.add(domain);
    return true;
  }

  /**
   * @param {string} domain canonical domain
   * @returns {boolean} true if removed, false if it was not present
   */
  remove(domain) {
    return this.domains.delete(domain);
  }

  /**
   * A domain is considered blacklisted if it matches exactly or is a subdomain
   * of a blacklisted entry (e.g. "mail.evil.com" matches a rule for "evil.com").
   * @param {string} domain canonical domain
   * @returns {boolean}
   */
  isBlacklisted(domain) {
    if (this.domains.has(domain)) return true;
    for (const blocked of this.domains) {
      if (domain.endsWith('.' + blocked)) return true;
    }
    return false;
  }
}
