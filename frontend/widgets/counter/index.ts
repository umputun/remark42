export function createCounter(target: HTMLElement, pageId: string) {}

export function mapConunters(selector: string, attributeName: string) {
  const counters = document.querySelectorAll(selector);

  for (let counter of Array.from(counters)) {
    const pageId = counter.getAttribute(attributeName ?? 'data-page-id');

    if (!(counter instanceof HTMLElement)) {
      console.error('Counter is not an HTMLElement', counter);
      continue;
    }
    if (pageId === null) {
      console.error(`Can't read pageId from ${attributeName}`);
      continue;
    }

    createCounter(counter, pageId);
  }
}
