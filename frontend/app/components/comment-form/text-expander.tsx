import '@github/text-expander-element';
import { h, Fragment, render, FunctionalComponent } from 'preact';
import { useEffect, useRef } from 'preact/hooks';
import cx from 'classnames';

import { StaticStore } from 'common/static-store';
import { Theme } from 'common/types';
import useTheme from 'hooks/useTheme';

import styles from './text-expander.module.css';

type Emoji = {
  key: string;
  emoji: string;
};

function SuggestionList({ items, theme }: { items: Array<Emoji>; theme: Theme }) {
  const isDarkTheme = theme === `dark`;
  const suggesterClass = cx(styles.suggester, { [styles.suggesterDark]: isDarkTheme });
  const suggesterItemClass = cx(styles.suggesterItem, { [styles.suggesterItemDark]: isDarkTheme });
  return (
    <ul className={suggesterClass}>
      {items.map(({ key, emoji }: Emoji) => {
        return (
          // eslint-disable-next-line jsx-a11y/role-has-required-aria-props
          <li role="option" className={suggesterItemClass} data-value={key}>
            <span className={styles.emojiResult}>{emoji}</span> {key}
          </li>
        );
      })}
    </ul>
  );
}

function searchEmoji(key: string, text: string, theme: Theme) {
  return import(/* webpackChunkName: "node-emoji" */ `node-emoji`)
    .then((nodeEmoji) => {
      if (key === ':') {
        const emojiList = nodeEmoji.search(text);
        if (emojiList.length === 0) {
          return Promise.resolve({ matched: false });
        }
        const fragment = document.createDocumentFragment();
        render(<SuggestionList theme={theme} items={emojiList.slice(0, 5)} />, fragment);
        return Promise.resolve({ matched: true, fragment: fragment.firstChild });
      }
      return Promise.resolve({ matched: false });
    })
    .catch(() => Promise.resolve({ matched: false }));
}

type ChangeListerEvent = Event & {
  detail: {
    key: string;
    text: string;
    provide(value: Promise<{ matched: boolean }>): void;
  };
};

type ValueListerEvent = Event & {
  detail: {
    key: string;
    value: string;
    item: HTMLLIElement;
  };
};

export const TextExpander: FunctionalComponent = ({ children }) => {
  const expanderRef = useRef<HTMLElement>();
  const theme = useTheme();
  useEffect(() => {
    if (expanderRef.current) {
      const expander = expanderRef.current;
      expander.setAttribute(`keys`, ':');
      const textExpanderChangeLister: EventListener = (event: Event) => {
        const { provide, key, text } = (event as ChangeListerEvent).detail;
        provide(searchEmoji(key, text, theme));
      };
      const textExpanderValueListener = (event: Event) => {
        const { key, item } = (event as ValueListerEvent).detail;
        if (key === ':') {
          (event as ValueListerEvent).detail.value = `:${item.dataset.value}:`;
        }
      };
      expander.addEventListener('text-expander-change', textExpanderChangeLister);
      expander.addEventListener('text-expander-value', textExpanderValueListener);
      return () => {
        expander.removeEventListener('text-expander-change', textExpanderChangeLister);
        expander.removeEventListener('text-expander-value', textExpanderValueListener);
      };
    }
    return () => undefined;
  }, [theme]);
  if (StaticStore.config.emoji_enabled) {
    return <text-expander ref={expanderRef}>{children}</text-expander>;
  }

  return <>{children}</>;
};
