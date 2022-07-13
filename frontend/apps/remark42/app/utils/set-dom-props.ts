export type StylesDeclaration = Partial<
  Record<
    keyof Omit<
      CSSStyleDeclaration,
      'length' | 'parentRule' | 'setProperty' | 'removeProperty' | 'item' | 'getPropertyValue' | 'getPropertyPriority'
    >,
    string | number
  >
>;

export function setStyles(element: HTMLElement, styles: StylesDeclaration = {}) {
  const entr = Object.entries(styles);

  entr.forEach(([p, v]) => {
    element.style[p as keyof StylesDeclaration] = `${v}`;
  });
}

export function setAttributes(element: HTMLElement, attrs: Record<string, string | number> = {}) {
  Object.entries(attrs).forEach(([p, v]) => {
    element.setAttribute(p, `${v}`);
  });
}
