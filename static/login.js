function getCookie(name) {
  const v = document.cookie.match('(^|;) ?' + name + '=([^;]*)(;|$)');
  return v ? v[2] : null;
}

const handleSignedInUser = function(user) {

	// Show redirection notice.
	document.getElementById('redirecting').style.visibility = 'visible';
	// Set session cookie
	user.getIdToken().then(function(idToken) {
		// Session login endpoint is queried and the session cookie is set.
		// CSRF token should be sent along with request.
		const csrfToken = getCookie('csrfToken')
		return postIdTokenToSessionLogin('/sessionLogin', idToken, csrfToken)
			.then(function() {
				// Redirect to profile on success.
				window.location.assign('/home');
			}, function(error) {
				// Refresh page on error.
				// In all cases, client side state should be lost due to in-memory
				// persistence.
				console.log(error);
				window.location.assign('/login');
			});
	});
};

const postIdTokenToSessionLogin = function(url, idToken, csrfToken) {
	// POST to session login endpoint.
	 return $.ajax({
		     type:'POST',
		     url: url,
		     data: {idToken: idToken, csrfToken: csrfToken},
		     contentType: 'application/x-www-form-urlencoded'
		   });
};

function getUiConfig() {
	return {
		'callbacks': {
			'signInSuccessWithAuthResult': function(authResult, redirectUrl) {
				handleSignedInUser(authResult.user);
				return false;
			},
			'uiShown': function() {
				document.getElementById('loading').style.display = 'none';
			}
		},
		'signInFlow': 'popup',
		'signInSuccessUrl': '/home',
		'signInOptions': [
			{
				provider: firebase.auth.EmailAuthProvider.PROVIDER_ID,
				requireDisplayName: true,
				signInMethod: "password",
			},
			firebase.auth.GoogleAuthProvider.PROVIDER_ID,
			firebase.auth.FacebookAuthProvider.PROVIDER_ID,
			firebase.auth.PhoneAuthProvider.PROVIDER_ID,
		],
	};
}

// get auth status
const initApp = function() {
	// Initialize the FirebaseUI Widget using Firebase.
	var ui = new firebaseui.auth.AuthUI(firebase.auth());
	ui.start('#firebaseui-auth-container', getUiConfig());
};

window.addEventListener('load', function() {
	initApp()
});

// Initialize Firebase
firebase.initializeApp(firebaseConfig);
//firebase.auth().setPersistence(firebase.auth.Auth.Persistence.NONE);
firebase.analytics();


