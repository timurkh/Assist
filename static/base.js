const globalMixin = {
	data() {
		return {
			loading:true,
			error_message:"",
		}
	},
	methods: {
		getStatusText : function(status) {

			switch(status) {
				case 0:
					return "Pending Approve";
				case 1:
					return "Member";
				case 2:
					return "Admin";
				case 3:
					return "Owner";
			};
		},
		getAxiosErrorMessage : function(error) {
			if (error.response != null && error.response.data != null) {
				return error.response.data;

			} else {
				return error;
			}
		},
	},
};

const csrfToken = document.getElementsByName("gorilla.csrf.Token")[0].value;

const postSignOut = function() {
	// POST to session login endpoint.
	axios({
		method: 'POST',
		url: '/sessionLogout',
		headers: { "X-CSRF-Token": csrfToken },
	})
	.then(function() {
		// Redirect to profile on success.
		window.location.assign('/login');
	}, function(error) {
		// Refresh page on error.
		// In all cases, client side state should be lost due to in-memory
		// persistence.
		console.log(error);
	});
}

// Initialize Firebase
firebase.initializeApp(firebaseConfig);
firebase.analytics();
