/** @jsx createElement */
import { createElement, RenderableProps, Fragment } from 'preact';
import { StaticStore } from '@app/common/static_store';
import { useEffect, useRef } from 'preact/hooks';
import '@github/text-expander-element';

function find(key: string, text: string) {
  return (
    import(/* webpackChunkName: "node-emoji" */ `node-emoji`)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .then((nodeEmoji: any) => {
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
          return Promise.resolve({ matched: true, fragment: menu });
        }
        return Promise.resolve({ matched: false });
      })
      .catch(() => Promise.resolve({ matched: false }))
  );
}

export function TextExpander({ children }: RenderableProps<void>) {
  const expanderRef = useRef<HTMLElement>();
  useEffect(() => {
    if (expanderRef.current) {
      const expander = expanderRef.current;
      expander.setAttribute(`keys`, ':');
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expander.addEventListener('text-expander-change', (event: any) => {
        const { provide, key, text } = event.detail;
        provide(find(key, text));
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
  if (StaticStore.config.emoji_enabled) {
    return <text-expander ref={expanderRef}>{children}</text-expander>;
  }

  return <Fragment>{children}</Fragment>;
}
