import { h, type FunctionComponent } from 'preact';
import styles from './counter.module.css';

export const Counter: FunctionComponent = ({ children }) => {
  return (
    <div className={styles.container} data-testid="comments-counter">
      {children}
    </div>
  );
};
