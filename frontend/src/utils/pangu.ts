import pangu from 'pangu/browser';

/**
 * Initialize automatic Pangu spacing across the page.
 * Uses MutationObserver to auto-space dynamically added or updated text nodes.
 */
export function initPangu(): void {
  if (typeof window === 'undefined') return;

  const start = () => {
    try {
      pangu.spacePage();
      pangu.autoSpacePage({
        pageDelayMs: 50,
        nodeDelayMs: 50,
        nodeMaxWaitMs: 200,
      });
    } catch (e) {
      console.warn('Failed to start pangu autoSpacePage:', e);
    }
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', start, { once: true });
  } else {
    start();
  }
}

/**
 * Format plain text with Pangu spacing (CJK + Western/Digits).
 */
export function spaceText(text: string): string {
  if (!text) return '';
  try {
    return pangu.spaceText(text);
  } catch {
    return text;
  }
}

/**
 * Immediately execute spacing on a specific DOM element or document.body.
 */
export function spacePage(node?: Node): void {
  if (typeof window === 'undefined') return;
  try {
    if (node) {
      pangu.spaceNode(node);
    } else {
      pangu.spacePage();
    }
  } catch (e) {
    console.warn('pangu spacing execution error:', e);
  }
}

export default pangu;
