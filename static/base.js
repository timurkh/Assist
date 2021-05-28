const { createApp } = Vue;

const globalMixin = {
	data() {
		return {
			loading:true,
			error_message:"",
			requestStatusesEnum : {
				WaitingApprove: 0,
				Processing: 1,
				Completed: 2,
				Declined: 3,
				Cancelled: 4,
			},
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
		getRequestStatusText:function(s) {
			switch(s) {
				case 0:
					return "Pending Approve";
				case 1:
					return "Being Processed";
				case 2:
					return "Completed";
				case 3:
					return "Declined";
			}
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

const notificationsToast = createApp( {
	data() {
		return {
			notifications: [],
			toastNotifications: [],
		}
	},
	delimiters: ['[[', ']]'],
	created:function() {
		// Load notifications
		axios({
			method: 'GET',
			url: `/methods/users/me/notifications`,
		})
		.then( res => {
			this.notifications = res.data;
		});
	},
	methods: {
		formatTime : function(time) {
			var t = new Date(time);
			return t.toLocaleString(undefined);
		},
		addNotification : function(n) {
			if(this.notifications != null)
				this.notifications.push(n);
			else
				this.notifications = [n];
		},
		showNotifications : function() {
			this.toastNotifications = [...this.notifications];
			this.$nextTick(() => {
				$('.toast').toast('show');
				axios({
					method: 'PUT',
					url: '/methods/users/me/notifications',
					headers: { "X-CSRF-Token": csrfToken},
				}).then(function () {
					this.notifications = [];
				}, function(error) {
					console.log(error);
				});
			});
		},
	},
}).mount("#notifications");

const navbar = createApp( {
	data() {
		return {
			notificationsCount: notificationsCount,
			notificationsEnabled: localStorage.getItem('notificationsEnabled'),
			messaging: {},
		}
	},
	delimiters: ['[[', ']]'],
	created:function() {
		// Initialize Firebase
		firebase.initializeApp(firebaseConfig);
		firebase.analytics();

		// Register service worker
		navigator.serviceWorker.register('/static/firebase-messaging-sw.js')
		.then((registration) => {
			if(devMode)
				console.log("Service worker registered");
			this.messaging = firebase.messaging();
			this.messaging.useServiceWorker(registration);
			this.catchMessages(this.messaging);	

			if ( this.notificationsEnabled) {
				if(Notification.permission === 'granted') {
					this.setupNotifications();
				} else {
					this.notificationsEnabled = false;
				}
			}

			// Listen to messages from the service worker
			navigator.serviceWorker.addEventListener('message', event => {
				// do not handle messages sent from firebase
				if(event.data != null && !event.data.isFirebaseMessaging) {
					if(devMode)
						console.log("SW event listener:", event);
					this.notificationsCount = event.data.count;
					notificationsToast.addNotification(event.data);
				}
			});
		})
		.catch( err => {
			this.notificationsEnabled = false;
			console.log(err);
		});

	},
	methods : {
		toggleNotifications:function() {
			localStorage.notificationsEnabled = this.notificationsEnabled;
			if (this.notificationsEnabled) {
				this.notificationsEnabled = false;
				Notification.requestPermission().then((permission) => {
					if (permission === 'granted') {
						this.setupNotifications();
					} else {
						if(devMode)
							console.log("user did not grant permissions to recieve notifications");
					}
				});
			}
			else {
				this.messaging.getToken().then((currentToken) => {
					this.messaging.deleteToken(currentToken)
					.catch( err => {
						console.log('Failed to delete FCM token: ', err);
					});
					axios({
						method: 'DELETE',
						url: `/methods/users/me/notifications`,
						headers: { "X-CSRF-Token": csrfToken },
					})
					.catch( err => {
						console.log("Failed to unsubscribe from notifications: ", err);
					});
				})
			}
		},
		initMessaging:function() {
		},
		setupNotifications:function() {
			// Send token to server
			this.messaging.getToken({vapidKey: 'BI4lx3GzJJfqbuv6COQ64ZIQcV5pBjTEMBAVby6ynjXrZV6D5FH8WEcpfWnm6a8z83brLRMo26QghpbShMygscc'})
			.then((currentToken) => {
				if (currentToken) {
					this.notificationsEnabled = true;
					if (messagingToken != currentToken) {
						if(devMode)
							console.log('Subscribing to firebase cloud messaging notifications with token ' + currentToken);
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
						if(devMode)
							console.log('Backend is already aware of this client token ' + currentToken);
					}
				}
			})
			.catch( err => {
				console.log(err);
			});
		},
		catchMessages:function(messaging) {
			// ok, success!
			messaging.onMessage((payload) => {
				console.log('Message received: ', payload);
				this.notificationsCount = payload.data.count;
				notificationsToast.addNotification(payload.data);
				navigator.serviceWorker.getRegistration('/static/firebase-messaging-sw.js').then((registration) => {
					const notificationOptions = {
						body: payload.data.text,
						icon: '/static/favicon.ico',
					};
					registration.showNotification(payload.data.title, notificationOptions);
				});
			});
		},
		showNotifications : function() {
			this.notificationsCount = 0;
			notificationsToast.showNotifications();
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
				console.log(error);
			});
		},
	},
	mixins: [globalMixin],
}).mount("#navbar");

const csrfToken = document.getElementsByName("gorilla.csrf.Token")[0].value;

