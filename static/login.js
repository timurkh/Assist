const { createApp } = Vue

const app = createApp( {
	delimiters: ['[[', ']]'],
	data(){
		return {
			loading:true,
			error_message:"",
			fire_ui:[],
		};
	},
	created:function() {
		// Initialize the FirebaseUI Widget using Firebase.
		fire_ui = new firebaseui.auth.AuthUI(firebase.auth());
		fire_ui.start('#firebaseui-auth-container', this.getUiConfig());
		this.loading = false;
	},
	methods: {
		handleSignedInUser:function(user) {

			this.loading = true;
			// Set session cookie
			user.getIdToken().then(idToken => {
				// Session login endpoint is queried and the session cookie is set.
				// CSRF token should be sent along with request.
				return this.postIdTokenToSessionLogin('/sessionLogin', idToken)
					.then(function() {
						// Redirect to profile on success.
						window.location.assign('/home');
					}, function(error) {
						this.error_message = ("Failed to login - " + error);
					});
			});
		},
		postIdTokenToSessionLogin : function(url, idToken) {
			const params = new URLSearchParams();
			params.append('idToken', idToken);
			return axios({
				method: 'POST',
				url: url,
				data: params,
				headers: { 
					'Content-Type': 'application/x-www-form-urlencoded',
					'X-CSRF-Token': csrfToken, },
			})
		},
		getUiConfig : function() {
			return {
				'callbacks': {
					'signInSuccessWithAuthResult': (authResult, redirectUrl) => {
						this.handleSignedInUser(authResult.user);
						return false;
					},
					'uiShown': function() {
						this.loading = false;
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
					{
						provider: firebase.auth.PhoneAuthProvider.PROVIDER_ID,
						defaultCountry: 'RU',
					},
				],
			};
		},
	},
}).mount("#app");

