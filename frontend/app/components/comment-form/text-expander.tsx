/** @jsx createElement */
import { createElement, RenderableProps } from 'preact';
import { useEffect, useRef } from 'preact/hooks';
import nodeEmoji from 'node-emoji';
import '@github/text-expander-element';

export function TextExpander({ children }: RenderableProps<void>) {
  const expanderRef = useRef<HTMLElement>();
  useEffect(() => {
    if (expanderRef.current) {
      const expander = expanderRef.current;
      expander.setAttribute(`keys`, ':');
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expander.addEventListener('text-expander-change', (event: any) => {
        const { key, provide, text } = event.detail;
        if (key === ':') {
          const emojiList = nodeEmoji.search(text);
          if (emojiList.length === 0) {
            return;
          }
          const menu = document.createElement('ul');
          menu.classList.add('suggester-container');
          menu.classList.add('suggester');
          menu.style.fontSize = `14px`;
          for (let i = 0; i < 5; i++) {
            const emoji = emojiList[i];
            if (emoji) {
              const item = document.createElement('li');
              item.setAttribute('role', 'option');
              item.dataset.emojiKey = emoji.key;
              item.textContent = emoji.emoji + ` ` + emoji.key;
              menu.append(item);
            }
          }

          provide(Promise.resolve({ matched: true, fragment: menu }));
        }
      });
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expander.addEventListener('text-expander-value', (event: any) => {
        const { key, item } = event.detail;
        if (key === ':') {
          event.detail.value = `:${item.dataset.emojiKey}:`;
        }
      });
    }
  }, []);
  return <text-expander ref={expanderRef}>{children}</text-expander>;
}
