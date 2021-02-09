const globalMixin = {
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
			}
		},
	},
};

const csrfToken = document.getElementsByName("gorilla.csrf.Token")[0].value;

const postSignOut = function() {
	// POST to session login endpoint.
	console.log(csrfToken);
	$.ajax({
		type:'POST',
		url: "/sessionLogout",
		headers: { 
			'X-CSRF-Token': csrfToken, },
	})
	.then(function() {
		window.location.assign('/login');
	}, function(error) {
		console.log(error);
	});
}

// Initialize Firebase
firebase.initializeApp(firebaseConfig);
firebase.analytics();
