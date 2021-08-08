import { h, JSX } from 'preact';

export function Arrow(props: JSX.HTMLAttributes<SVGSVGElement>) {
  return (
    <svg width="14" height="14" viewBox="0 0 28 28" fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
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
