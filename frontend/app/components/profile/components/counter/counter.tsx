import { h } from 'preact';
import styles from './counter.module.css';

export const Counter: React.FC = ({ children }) => {
  return <div className={styles.container}>{children}</div>;
};
