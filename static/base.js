Vue.mixin({
	methods: {
		getStatusText : function(status) {

			switch(status) {
				case 0 /*PendingApprove*/:
					return "Pending Approve";
				case 1 /*Member*/:
					return "Member";
				case 2 /*Admin*/:
					return "Admin";
				case 3 /*Owner*/:
					return "Owner";
			}
		},
	},
})

const postSignOut = function() {
	// POST to session login endpoint.
	$.ajax({
		type:'POST',
		url: '/sessionLogout',
		contentType: 'application/x-www-form-urlencoded'
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
