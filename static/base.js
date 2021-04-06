const { createApp } = Vue

const globalMixin = {
	data() {
		return {
			loading:true,
			error_message:"",
		}
	},
	methods: {
		getDate : function(date) {
			return date.toLocaleString('ru', {
				    day:   '2-digit',
				    month: '2-digit',
				    year:  '2-digit'
				  });
		},
		getEventStatusText : function(status) {

			switch(status) {
				case 0:
					return "Not going";
				case 1:
					return "Applied";
				case 2:
					return "Going";
				case 3:
					return "Attended";
				case 4:
					return "No-show";
			};
		},
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
			if (error.response != null && error.response.data != null && error.response.data != "") {
				return error.response.data;

			} else {
				return error;
			}
		},
		getTagHasValues : function(tag) {
			if (tag == null || tag.values == null)
				return false;
			let valuesCount = Object.entries(tag.values).length;
			return valuesCount > 1 || (valuesCount == 1 && tag.values._ == null);
		},
		getTagValues : function(tags) {
			let tagValues = [];
			let i=0;
			while(tags.length>i) {
				let tag = tags[i];
				values = Object.entries(tag.values);
				if (values.length > 1) {
					for (const [key, value] of values) {
						tagValues.push(tag.name + "/" + key);
					}
				} else {
					tagValues.push(tag.name);
				}
				i++;
			}
			return tagValues;
		},
	},
};

const navbar = createApp( {
	data() {
		return {
			notificationsCount: notificationsCount,
			notificationsEnabled: false,
			notificationsFailure: false,
			messaging: {},
		}
	},
	delimiters: ['[[', ']]'],
	created:function() {
		// Initialize Firebase
		firebase.initializeApp(firebaseConfig);
		firebase.analytics();

		let ne = localStorage.getItem('notificationsEnabled');
		this.notificationsEnabled = (ne == 'true');
		
		this.initMessaging();
	},
	methods : {
		toggleNotifications:function() {
			localStorage.notificationsEnabled = this.notificationsEnabled;
			if (this.notificationsEnabled) {
				this.notificationsFailure = true;
				Notification.requestPermission().then((permission) => {
					if (permission === 'granted') {
						this.setupNotifications();
					} else {
						console.log("user did not grant permissions to recieve notifications");
					}
				});
			}
			else
				this.notificationsFailure = false;
		},
		initMessaging:function() {
			this.messaging = firebase.messaging();

			// Register service worker
			navigator.serviceWorker.register('/static/firebase-messaging-sw.js')
			.then((registration) => {
				this.messaging.useServiceWorker(registration);
				this.catchMessages(this.messaging);	
				if (this.notificationsEnabled)
					this.setupNotifications();
			});
		},
		setupNotifications:function() {
			// Send token to server
			console.log("getting token");
			this.messaging.getToken({vapidKey: 'BI4lx3GzJJfqbuv6COQ64ZIQcV5pBjTEMBAVby6ynjXrZV6D5FH8WEcpfWnm6a8z83brLRMo26QghpbShMygscc'})
			.then((currentToken) => {
				console.log("messagingToken = " + currentToken);
				if (currentToken) {
					if (messagingToken != currentToken) {
						console.log('Sending token ' + currentToken + ' to server...');
						axios({
							method: 'POST',
							url: `/methods/users/me/notifications`,
							data: {
								token: currentToken,
							},
							headers: { "X-CSRF-Token": csrfToken },
						})
						.then( res => {
							this.catchMessages(this.messaging);	
						});
					} else {
						this.catchMessages(this.messaging);					
					}
				}
			})
			.catch( err => {
				console.log(err);
			});
		},
		catchMessages:function(messaging) {
			// ok, success!
			this.notificationsFailure = false;
			messaging.onMessage((payload) => {
				console.log('Message received. ', payload);
				this.notificationsCount = payload.data.count;

				new Notification('There are ' + payload.data.count + ' new notifications', { body: payload.data.text, icon: '/favicon.ico' });
			});
		},
		postSignOut : function() {
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
		},
	},
	mixins: [globalMixin],
}).mount("#navbar");

const csrfToken = document.getElementsByName("gorilla.csrf.Token")[0].value;

