if (typeof pb === 'undefined') {
	window.pb = new PocketBase();
}

function initAuth() {
	if (pb.authStore.isValid) {
		// Suppose you’ve already logged in:
		const rawToken = pb.authStore.token;

		// Check if the current environment uses HTTPS
		const isSecure = window.location.protocol === 'https:';
		document.cookie = pb.authStore.exportToCookie({
			httpOnly: false,
			secure: isSecure,
			sameSite: isSecure ? 'None' : 'Lax',
		});

		// Now set a cookie with only the token as its value
		htmx.on('htmx:configRequest', (e) => {
			e.detail.headers['Authorization'] = rawToken;
		});
	}
}

async function createUser(email, passInput, passConfirm) {
	try {
		const data = {
			"email": email,
			"password": passInput,
			"passwordConfirm": passConfirm,
		};
		const record = await pb.collection('users').create(data);
		return record;
	} catch (error) {
		throw error;
	}
}

async function loginUser(email, passInput) {
	try {
		const authData = await pb.collection('users').authWithPassword(email, passInput);
		initAuth();
		return authData;
	} catch (error) {
		throw error;
	}
}

async function loginOAuth(provider) {
	try {
		const authData = await pb.collection('users').authWithOAuth2({ provider: provider });
		initAuth();
		return authData;
	} catch (error) {
		throw error;
	}
}

async function logout() {
	try {
		await pb.authStore.clear();
		document.cookie = `pb_auth=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
	} catch (error) {
		throw error;
	}
}

initAuth();
