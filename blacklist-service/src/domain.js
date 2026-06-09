// Domain parsing, validation and normalisation.

// Reasonable hostname pattern: labels of letters/digits/hyphens separated by dots, with a final alphabetic TLD of at least two characters.
// Rejects schemes, paths, ports, spaces and IP addresses.
const DOMAIN_RE =
  /^(?=.{1,253}$)([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/;

export class InvalidDomainError extends Error {
  constructor(message) {
    super(message);
    this.name = 'InvalidDomainError';
  }
}

/**
 * Normalise and validate a user-supplied domain string.
 * Accepts inputs like "https://www.Example.com/path" and returns "example.com".
 * @param {unknown} input
 * @returns {string} the canonical lowercase hostname
 * @throws {InvalidDomainError}
 */
export function normalizeDomain(input) {
  if (typeof input !== 'string') {
    throw new InvalidDomainError('domain must be a string');
  }

  let value = input.trim().toLowerCase();
  if (value === '') {
    throw new InvalidDomainError('domain must not be empty');
  }

  // If a full URL was supplied, extract just the hostname.
  if (value.includes('://')) {
    try {
      value = new URL(value).hostname;
    } catch {
      throw new InvalidDomainError('domain is not a valid URL or hostname');
    }
  }

  // Strip a leading "www." so the rule applies to the registrable domain.
  if (value.startsWith('www.')) {
    value = value.slice(4);
  }

  // Drop a trailing dot (fully-qualified form).
  if (value.endsWith('.')) {
    value = value.slice(0, -1);
  }

  if (!DOMAIN_RE.test(value)) {
    throw new InvalidDomainError('domain format is invalid');
  }

  return value;
}
