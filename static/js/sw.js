self.addEventListener('push', event => {
    console.log('[Service Worker] Push Received.');
    console.log(`[Service Worker] Push had this data: "${event.data.text()}"`);

    const title = 'Neues Maimai postiert!';
    const options = {
        body: event.data.text(),
    };

    event.waitUntil(self.registration.showNotification(title, options));
});

self.onnotificationclick = function (event) {
    event.notification.close();

    // This looks to see if the current is already open and
    // focuses if it is
    event.waitUntil(clients.matchAll({
        type: "window"
    }).then(function (clientList) {
        if (clients.openWindow)
            return clients.openWindow('/');
    }));
};