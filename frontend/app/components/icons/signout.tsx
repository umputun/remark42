import { h, JSX } from 'preact';

type Props = Omit<JSX.SVGAttributes<SVGSVGElement>, 'size'> & {
  size?: number | string;
};

export function SignOutIcon({ size = 16, ...props }: Props) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width={size} height={size} viewBox="0 0 16 16" fill="none" {...props}>
      <path
        stroke="currentColor"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="1.5"
        d="M5.7 15H2.6A1.6 1.6 0 011 13.4V2.6A1.6 1.6 0 012.6 1h3M11.1 11.9L15 8l-3.9-3.9M15 8H5.7"
      />
    </svg>
  );
}
