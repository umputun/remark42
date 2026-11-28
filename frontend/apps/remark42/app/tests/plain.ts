/**
 * Draws a component written in plain JavaScript into the page for a test, and destroys it and
 * takes it off the page after the test, as the testing library does with what it renders.
 */

type Part = { destroy(): void };

const drawn: Array<{ container: HTMLElement; part: Part }> = [];

afterEach(() => {
  drawn.splice(0).forEach(({ container, part }) => {
    part.destroy();
    container.remove();
  });
});

/**
 * Mounts a part into a new element of this tag, inside a container on the page.
 *
 * @returns the container, whose innerHTML is the part's own element, the element and the part
 */
export function draw<T extends Part, K extends keyof HTMLElementTagNameMap>(
  tag: K,
  mount: (root: HTMLElementTagNameMap[K]) => T
) {
  const container = document.createElement('div');
  const root = document.createElement(tag);
  container.append(root);
  document.body.append(container);
  const part = mount(root);
  drawn.push({ container, part });
  return { container, root, part };
}
