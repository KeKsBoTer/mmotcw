function inView(element) {
    // get window height
    var windowHeight = window.innerHeight;
    var elementHeight = element.clientHeight;
    // get number of pixels that the document is scrolled
    var scrollY = window.scrollY || window.pageYOffset;

    // get current scroll position (distance from the top of the page to the bottom of the current viewport)
    var scrollPosition = scrollY + windowHeight;
    // get element position (distance from the top of the page to the bottom of the element)
    var elementPosition = element.getBoundingClientRect().top + scrollY + elementHeight;

    // is scroll position greater than element position? (is element in view?)
    if (scrollPosition > elementPosition) {
        return true;
    }

    return false;
}


// render Charts as soon as they come into view
var observer = new IntersectionObserver(function (entries) {
    for (let { isIntersecting, target } of entries)
        if (isIntersecting) {
            this.unobserve(target);
            let container = target;
            let data = JSON.parse(atob(container.getAttribute("data")));
            var chart = new CanvasJS.Chart(container, {
                animationEnabled: true,
                backgroundColor: "transparent",
                data: [{
                    type: "pie",
                    yValueFormatString: "#,##\"\"",
                    toolTipContent: "{label} - #percent % ({y})",
                    indexLabel: "{label}",
                    dataPoints: data.map(x => {
                        return {
                            label: x.FileName.split(".")[0].split("_").slice(1, 3).join("_"),
                            y: x.Votes
                        }
                    })
                }]
            });
            chart.render();
            container.querySelector(".canvasjs-chart-credit").remove();
        }
},
    { threshold: [0] }
);

document.querySelectorAll(".chartContainer").forEach(e => observer.observe(e));

window.onload = function () {
    var elevator = new Elevator({
        element: document.querySelector('.elevator-button'),
        mainAudio: 'static/sound/elevator.mp3',
        endAudio: 'static/sound/ding.mp3'
    });
    for (var img of document.getElementsByClassName('maimai')) {
        if (img.complete) {
            img.style.setProperty("background-image", "none")
            img.style.setProperty("filter", "none")
        } else {
            img.addEventListener('load', (evt) => {
                evt.target.style.setProperty("background-image", "none")
                evt.target.style.setProperty("filter", "none")
            })
        }
    }
}