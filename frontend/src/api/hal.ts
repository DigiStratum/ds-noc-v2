/**
 * HAL+JSON types and helpers for HATEOAS discovery
 * @see https://datatracker.ietf.org/doc/html/draft-kelly-json-hal
 */

/**
 * A HAL link object
 */
export interface HALLink {
  href: string;
  templated?: boolean;
  title?: string;
  type?: string;
  deprecation?: string;
  name?: string;
  profile?: string;
  hreflang?: string;
}

/**
 * Collection of HAL links keyed by relation
 */
export interface HALLinks {
  [rel: string]: HALLink | HALLink[];
}

/**
 * A CURIE (Compact URI) for link relation documentation
 */
export interface HALCurie {
  name: string;
  href: string;
  templated: boolean;
}

/**
 * Base HAL resource with optional _links and _embedded
 */
export interface HALResource {
  _links?: HALLinks;
  _embedded?: Record<string, HALResource | HALResource[]>;
}

/**
 * Root discovery response from /api/discovery
 */
export interface DiscoveryResponse extends HALResource {
  service: string;
  version: string;
}

/**
 * Extract a single link from HAL links (handles array case)
 */
export function getLink(links: HALLinks | undefined, rel: string): HALLink | undefined {
  if (!links) return undefined;
  const link = links[rel];
  if (Array.isArray(link)) {
    return link[0];
  }
  return link;
}

/**
 * Get the href from a HAL link, expanding templated URIs if values provided
 */
export function getHref(
  links: HALLinks | undefined,
  rel: string,
  params?: Record<string, string>
): string | undefined {
  const link = getLink(links, rel);
  if (!link) return undefined;

  let href = link.href;
  if (link.templated && params) {
    // Simple URI template expansion (handles {param} style)
    Object.entries(params).forEach(([key, value]) => {
      href = href.replace(`{${key}}`, encodeURIComponent(value));
    });
  }
  return href;
}

/**
 * Check if a resource has a specific link relation
 */
export function hasLink(links: HALLinks | undefined, rel: string): boolean {
  return links !== undefined && rel in links;
}
