function createElement(message) {
	const el = document.createElement('span');
	el.innerHTML = message;
	el.classList.add(
		'animate-from-right-36',
		'bg-brand-100',
		'dark:bg-brand-900',
		'dark:text-brand-400',
		'dark:text-gray-300',
		'fixed',
		'font-medium',
		'px-2',
		'py-1',
		'rounded',
		'shadow',
		'text-brand-600',
		'text-center',
		'w-36',
		'z-10',
		window.innerWidth > '768' ? 'top-20' : 'top-24',
	);
	el.setAttribute('role', 'alert');

	document.body.after(el);

	setTimeout(() => {
		el.classList.remove('animate-from-right-36');
		el.classList.add('animate-to-right-36');
		setTimeout(() => {
			el.remove();
		}, 300);
	}, 3000);
}

function showSuccess() {
	createElement('Copied success');
}

function showError() {
	createElement('Copied error');
}

function getURL(el) {
	const url = window.location.href.replace(window.location.hash, '');
	return `${url}#${el.parentElement.id}`;
}

function handleCopy(el) {
	navigator.clipboard
		.writeText(getURL(el))
		.then(showSuccess)
		.catch(showError);
}

window.onload = () => {
	[...document.querySelectorAll('[data-permalink]')].forEach((el) => {
		el.addEventListener('click', () => handleCopy(el));
	});
};
