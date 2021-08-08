import { h, JSX } from 'preact';

type Props = { size?: number } & JSX.HTMLAttributes<SVGSVGElement>;

export function MdIcon({ size = 16, ...props }: Props) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
      {...props}
    >
      <path
        fill-rule="evenodd"
        clip-rule="evenodd"
        d="M22.789 4.5H2.226C1.28 4.5.5 5.28.5 6.226v11.542c0 .96.78 1.741 1.726 1.741h20.548c.96 0 1.726-.78 1.726-1.726V6.226A1.714 1.714 0 0022.789 4.5zm-8.781 12.007h-3.002v-4.503l-2.251 2.882-2.252-2.882v4.503H3.502V7.501h3.001l2.252 3.002 2.251-3.002h3.002v9.006zm4.489.75l-3.738-5.253h2.252V7.501h3.002v4.503h2.251l-3.767 5.253z"
      />
    </svg>
  );
}
