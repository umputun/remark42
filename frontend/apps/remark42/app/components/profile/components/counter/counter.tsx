import { h, type FunctionComponent } from 'preact';
import clsx from 'clsx';

import styles from './counter.module.css';

export const Counter: FunctionComponent = ({ children }) => {
  return (
    // the class is kept outside the css modules so the browser suite can read the count: the
    // production bundle hashes the module names and strips the data-testid beside it
    <div className={clsx('comments-counter', styles.container)} data-testid="comments-counter">
      {children}
    </div>
  );
};
