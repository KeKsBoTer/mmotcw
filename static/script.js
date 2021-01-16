window.onload = function () {
    var elevator = new Elevator({
        element: document.querySelector('.elevator-button'),
        mainAudio: 'static/sound/elevator.mp3',
        endAudio: 'static/sound/ding.mp3'
    });
    for (var img of document.getElementsByClassName('maimai')) {
        if (img.complete) {
            img.style.setProperty("background-image","none")
            img.style.setProperty("filter","none")
        } else {
            img.addEventListener('load', (evt) => {
                evt.target.style.setProperty("background-image","none")
                evt.target.style.setProperty("filter","none")
            })
        }
    }

}