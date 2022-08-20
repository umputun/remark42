<script lang="ts">
	import type { Sort } from '@remark42/api'
	import { t } from './lib/i18n'

	import { user, isLoading, logout } from './stores/user'
	import { addComment, comments, sort } from './stores/comments'
	import { fetchConfig } from './stores/config'

	import CommentsTree from './components/comments-tree.svelte'
	import CommentForm from './components/comment-form/comment-form.svelte'
	import PoweredByRemark42 from './components/powered-by-remark.svelte'
	import Auth from './components/auth/auth.svelte'
	import User from './components/user.svelte'
	import Button from './components/ui/button.svelte'
	import Icon from './components/ui/icons/icon.svelte'
	import Dropdown from './components/ui/dropdown.svelte'

	fetchConfig()

	let text = ''
	async function handleSubmitComment() {
		await addComment(text)
	}

	function handleChangeSort(evt: Event) {
		sort.set((evt.currentTarget as HTMLSelectElement).value as Sort)
	}
	function handleOpenAdminSettings() {}
</script>

<section class="root">
	<header class="header">
		<div class="flex justify-between">
			<User id={$user?.id} name={$user?.name} picture={$user?.picture}>
				<svelte:fragment slot="actions">
					{#if !$isLoading}
						<Dropdown>
							<svelte:fragment slot="button" let:onClick>
								<Button kind="seamless" title={$t('Subscribe to notifications')} on:click={onClick}>
									<Icon name="bell" />
								</Button>
							</svelte:fragment>

							Lorem ipsum dolor sit amet consectetur adipisicing elit. Vitae nulla similique culpa
							eum ad, amet accusamus totam debitis aliquam suscipit quam necessitatibus. Id odit
							placeat illum labore nulla eveniet libero.
						</Dropdown>
						{#if $user?.admin}
							<Button kind="seamless" on:click={handleOpenAdminSettings}>
								<Icon name="gear" />
							</Button>
						{/if}
						<Button kind="seamless" on:click={logout}>
							<Icon name="signout" />
						</Button>
					{/if}
				</svelte:fragment>
			</User>
			<div class="flex space-x-2 items-center">
				<Dropdown>
					<svelte:fragment slot="button" let:onClick>
						<Button kind="seamless" title={$t('Subscribe to RSS')} on:click={onClick}>
							<Icon name="rss" />
						</Button>
					</svelte:fragment>

					Lorem ipsum dolor sit, amet consectetur adipisicing elit. Nisi corrupti fugiat aliquid
					optio totam cum minima repudiandae sapiente, tempora quisquam quidem autem quis quibusdam
					vitae, dicta maxime mollitia sunt est.
				</Dropdown>
			</div>
		</div>

		<CommentForm
			bind:value={text}
			on:submit={handleSubmitComment}
			as={$user ? 'form' : 'div'}
			hideSubmitButton={!$user}
		>
			<svelte:fragment slot="actions">
				{#if !$user}
					<Auth />
				{/if}
			</svelte:fragment>
		</CommentForm>
	</header>
	{#if $comments}
		<main>
			<select on:change={handleChangeSort} bind:value={$sort}>
				<option value="-score">{$t('-score')}</option>
				<option value="+score">{$t('+score')}</option>
				<option value="-time">{$t('-time')}</option>
				<option value="+time">{$t('+time')}</option>
				<option value="-active">{$t('-active')}</option>
				<option value="+active">{$t('+active')}</option>
				<option value="-controversy">{$t('-controversy')}</option>
				<option value="+controversy">{$t('+controversy')}</option>
			</select>
			<CommentsTree comments={$comments} />
		</main>
	{/if}
	<footer class="footer">
		<PoweredByRemark42 />
	</footer>
</section>

<style global lang="postcss">
	@tailwind base;
	@layer {
		textarea:focus {
			outline: none;
		}
	}
	@tailwind components;
	@tailwind utilities;

	:root {
		color-scheme: light dark;
	}

	.root {
		@apply text-sm text-slate-800 dark:text-slate-100 p-5 w-[500px];
	}

	.header {
		@apply mb-4;
	}
</style>
