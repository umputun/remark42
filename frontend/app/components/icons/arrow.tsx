import { h, JSX } from 'preact';

type Props = { size?: number } & JSX.HTMLAttributes<SVGSVGElement>;

export function ArrowIcon({ size = 14, ...props }: Props) {
  return (
    <svg width={size} height={size} viewBox="0 0 28 28" fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
      <path
        d="M6 11.5L14.5 19L22 11"
        stroke="currentColor"
        stroke-width="4"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}
