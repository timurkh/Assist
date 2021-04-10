importScripts("https://www.gstatic.com/firebasejs/8.3.2/firebase-app.js");
importScripts("https://www.gstatic.com/firebasejs/8.3.2/firebase-messaging.js");
importScripts("/static/firebase.js");

firebase.initializeApp(firebaseConfig);
const messaging = firebase.messaging();

console.log("Listening for FCM messages in ServiceWorker");

messaging.onBackgroundMessage(function(payload) {
	const notificationOptions = {
		body: payload.data.text,
		icon: '/static/favicon.ico',
		data: payload.data,
	};

	console.log("ServiceWorker recieved message: ", payload);

	// send message to open browser tabs
	clients.matchAll({
        type: 'window',
        includeUncontrolled: true
    }).then(function(clientList) {
        for (var i = 0; i < clientList.length; i++) {
            var client = clientList[i];
            if ('postMessage' in client) {
				client.postMessage(payload.data);
			}
		}
	});
	
	// create notification
	self.registration.showNotification(payload.data.title,
		notificationOptions);
});

self.addEventListener('notificationclick', function(event) {
    event.notification.close();

	// set focus on browser tab if exists, otherwise open new tab
    event.waitUntil(clients.matchAll({
        type: 'window',
        includeUncontrolled: true
    }).then(function(clientList) {
        for (var i = 0; i < clientList.length; i++) {
            var client = clientList[i];
            if ('focus' in client) {
                return client.focus();
            }
        }

        return clients.openWindow("/");
    }));
});
