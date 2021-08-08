import { h, JSX } from 'preact';

type Props = { size?: number } & JSX.HTMLAttributes<SVGSVGElement>;

export function RssIcon({ size = 16, ...props }: Props) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
      <path
        d="M4 11a9 9 0 019 9M4 4a16 16 0 0116 16M5 20a1 1 0 100-2 1 1 0 000 2z"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}
