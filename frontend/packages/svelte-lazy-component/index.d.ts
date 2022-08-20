import type { ComponentProps, ComponentType, SvelteComponent, SvelteComponentTyped } from 'svelte'

export default class LazyComponent<C extends SvelteComponent> extends SvelteComponentTyped<
	{
		shouldLoad: boolean | undefined
		loader: () => Promise<{ default: ComponentType<C> }>
	} & ComponentProps<C>
> {}
