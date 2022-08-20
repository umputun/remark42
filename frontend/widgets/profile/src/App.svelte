<script lang="ts">
	import type { User } from '@remark42/api'
	import { t } from '@remark42/common/i18n'
	import Button from '@remark42/common/ui/button.svelte'
	import Icon from '@remark42/common/ui/icons/icon.svelte'
	import Avatar from '@remark42/common/ui/avatar.svelte'
	import Spinner from '@remark42/common/ui/spinner.svelte'

	export let isCurrent = false
	export let user: User

	let isError
	let isLoading
	let isLoaded
	let isLoadMoreVisible
	let isSigningOut = false

	let comments = []

	function fetchComments() {}
	function handleClickLogout() {}
	function handleClickRequestRemoveData() {}
</script>

<section class="profile">
	<Button>
		<Icon name="cross" />
	</Button>

	<aside class="profile-sidebar" class:profile_current={isCurrent}>
		<header class="profile-header">
			<div class="profile-avatar">
				<Avatar src={user.picture} username={user.name} />
			</div>
			<div class="profile-content">
				<div class="profile-title">{user.name}</div>
				<div class="profile-id">{user.id}</div>
			</div>
			{#if isCurrent}
				<button
					class="profile-signout"
					title={$t('Sign out')}
					onClick={handleClickLogout}
					disabled={isSigningOut}
				>
					{#if isSigningOut}
						<Spinner />
					{:else}
						<Icon name="signout" />{/if}
				</button>
			{/if}
		</header>
		<section class="profile-content">
			<!-- Loading state -->
			{#if isLoading}
				<Spinner />
			{/if}

			<!-- Error state -->
			{#if isError}
				<div class="profile-error">
					<p class="profile-error-message">
						{$t('Something went wrong. Please try again a bit later.')}
					</p>
					<Button kind="seamless" on:click={fetchComments}>
						${t('Retry')}
					</Button>
				</div>
			{/if}

			<!-- Loaded state -->
			{#if isLoaded}
				{#if comments.length === 0}
					<p class="profile-emptyState">
						{t(`Don't have comments yet`)}
					</p>
				{:else}
					<h3 class="profile-title">
						{#if isCurrent}
							{$t('My comments')}
						{:else}
							{$t('Comments')}
						{/if}
						{#if comments.length}
              <div class="profile-commentsCounter" title={$t('Comments count')}>
                {comments.length}
              </div>
						{/if}
					</h3>
				{/if}

				{#each comments as comment}
					<div>{comment.user}</div>
				{/each}

				{#if isLoadMoreVisible}
					<Button kind="seamless" on:click={fetchComments}>
						{#if isLoading}
							<Spinner />
						{:else}
							{$t('Load more')}
						{/if}
					</Button>
				{/if}
			{/if}
		</section>
		{#if isCurrent}
			<footer class="profile-footer">
				<Button kind="seamless" on:click={handleClickRequestRemoveData}>
					{$t('Request my data removal')}
				</Button>
			</footer>
		{/if}
		<!--  TODO: implement hiding user comments -->
	</aside>
</section>

<style lang="postcss">
</style>
