import { h, JSX } from 'preact';

type Props = { size?: number } & JSX.HTMLAttributes<SVGSVGElement>;

export function ArrowIcon({ size = 16, ...props }: Props) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
      <path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  );
}
