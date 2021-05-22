import { h, JSX } from 'preact';

type Props = Omit<JSX.SVGAttributes<SVGSVGElement>, 'size'> & {
  size?: number | string;
};

export function CrossIcon({ size = 14, ...props }: Props) {
  return (
    <svg width={size} height={size} viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
      <path
        d="M2 2L12 12M12 2L2 12"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
}
