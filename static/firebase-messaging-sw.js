importScripts("https://www.gstatic.com/firebasejs/8.3.1/firebase-app.js");
importScripts("https://www.gstatic.com/firebasejs/8.3.1/firebase-messaging.js");
importScripts("/static/firebase.js");

firebase.initializeApp(firebaseConfig);
const messaging = firebase.messaging();

messaging.onBackgroundMessage(function(payload) {
  console.log('[firebase-messaging-sw.js] Received background message ', payload);
  // Customize notification here
  const notificationTitle = 'Background Message Title';
  const notificationOptions = {
    body: payload.notification.body,
    icon: '/static/favicon.ico',
	click_action: 'http://locahost:8080/home',
  };

  self.registration.showNotification(payload.notification.title,
    notificationOptions);
});
