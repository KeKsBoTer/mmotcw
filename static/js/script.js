window.onload = function () {
    new Elevator({
        element: document.querySelector('.elevator-button'),
        mainAudio: '/static/sound/elevator.mp3',
        endAudio: '/static/sound/ding.mp3'
    });
}

function subscribe(registration) {
    registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
    }).then(function (subscription) {
        fetch("subscribe", {
            method: "POST",
            body: JSON.stringify(subscription),
            headers: {
                "Content-type": "application/json; charset=UTF-8"
            }
        })
    }).catch(e=>console.error("cannot subscribe:",e))
}

function urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding)
        .replace(/\-/g, '+')
        .replace(/_/g, '/');
    const rawData = window.atob(base64);
    return Uint8Array.from([...rawData].map(char => char.charCodeAt(0)));
}

if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/static/js/sw.js').then(function (registration) {
        registration.pushManager.getSubscription().then((subscription) => {
            if (!subscription) {
                subscribe(registration);
            }
        });
    })
}