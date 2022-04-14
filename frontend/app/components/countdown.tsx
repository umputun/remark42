import { h, Fragment } from 'preact';
import { useEffect, useRef, useState } from 'preact/hooks';

type Props = {
  timestamp?: number;
  onTimePassed?: () => void;
};

export function Countdown({ timestamp = 0, onTimePassed }: Props) {
  const [value, setValue] = useState(calcRestTime(timestamp));
  const intervalIdRef = useRef<number | undefined>();

  useEffect(() => {
    if (!timestamp) {
      return;
    }

    const intervalId = window.setInterval(() => setValue(calcRestTime(timestamp || 0)), 1000);
    intervalIdRef.current = intervalId;

    setValue(calcRestTime(timestamp || 0));

    return () => {
      window.clearInterval(intervalId);
    };
  }, [timestamp]);

  useEffect(() => {
    if (value === 0) {
      onTimePassed?.();
      window.clearInterval(intervalIdRef.current);
    }
  }, [value, onTimePassed]);

  if (!timestamp) {
    return null;
  }

  return <Fragment>{value}s</Fragment>;
}

function calcRestTime(timestamp: number): number {
  return Math.ceil(Math.max(0, (timestamp - Date.now()) / 1000));
}
