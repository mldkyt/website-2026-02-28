function stars() {
    var canvas = document.getElementById("background");
    var context = canvas.getContext("2d");

    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;

    context.fillStyle = "black";

    var stars = [];

    function newStar() {
        for (var index = 0; index < moveAmount; index++) {
            stars.push({
                x: pageX,
                y: pageY,
                dir: Math.random() * Math.PI * 2,
            });
        }

        stars = stars.filter(function (x) {
            return x.x < canvas.width * 2 || x.y < canvas.height * 2;
        });
    }

    function draw() {
        context.fillStyle = "black";
        context.fillRect(0, 0, canvas.width, canvas.height);

        for (var index = 0; index < stars.length; index++) {
            var star = stars[index];
            star.x += Math.cos(star.dir) * 6;
            star.y += Math.sin(star.dir) * 6;
            context.fillStyle = "white";
            context.fillRect(Math.round(star.x / 2) * 2, Math.round(star.y / 2) * 2, 2, 2);
        }
    }

    function clearStars() {
        // remove stars that are off screen
        let stars_before = stars.length;
        stars = stars.filter(function (x) {
            return x.x > 0 && x.y > 0 && x.x < canvas.width && x.y < canvas.height;
        });
        let stars_after = stars.length;
        console.log(`Cleared ${stars_before - stars_after} stars`);
    }

    setInterval(newStar, 10);
    setInterval(clearStars, 1000);

    document.addEventListener("resize", function () {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    });

    setInterval(draw, 10);

    context.fillStyle = "black";
    context.fillRect(0, 0, canvas.width, canvas.height);

    for (var index = 0; index < 100; index++) {
        newStar();
        draw();
    }

    document.onscroll = function () {
        const scrollPos = window.scrollY;
        canvas.style.transform = `translateY(${scrollPos}px)`;
    };

    window.addEventListener("resize", function () {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    });

    var pageX = window.innerWidth / 2;
    var pageY = window.innerHeight / 2;
    var moveAmount = 0;
    var lastTimeout = null;

    (function () {
        document.onmousemove = handleMouseMove;
        function handleMouseMove(event) {
            var eventDoc, doc, body;

            event = event || window.event; // IE-ism

            // If pageX/Y aren't available and clientX/Y are,
            // calculate pageX/Y - logic taken from jQuery.
            // (This is to support old IE)
            if (event.pageX == null && event.clientX != null) {
                eventDoc = (event.target && event.target.ownerDocument) || document;
                doc = eventDoc.documentElement;
                body = eventDoc.body;

                event.pageX = event.clientX + ((doc && doc.scrollLeft) || (body && body.scrollLeft) || 0) - ((doc && doc.clientLeft) || (body && body.clientLeft) || 0);
                event.pageY = event.clientY + ((doc && doc.scrollTop) || (body && body.scrollTop) || 0) - ((doc && doc.clientTop) || (body && body.clientTop) || 0);
            }

            var lastPageX = pageX;
            var lastPageY = pageY;

            pageX = event.pageX - window.scrollX;
            pageY = event.pageY - window.scrollY;

            moveAmount = Math.abs(lastPageX - pageX) + Math.abs(lastPageY - pageY);
            moveAmount /= 5;

            if (lastTimeout != null) {
                clearTimeout(lastTimeout);
            }

            lastTimeout = setTimeout(function () {
                moveAmount = 0;
            }, 100);
        }
    })();
}

function starsStatic() {
    var canvas = document.getElementById("background");
    var context = canvas.getContext("2d");

    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
    

    function draw() {
        const rand = new Math.seedrandom('1');
    
        context.fillStyle = "black";
        context.fillRect(0, 0, canvas.width, canvas.height);

        context.fillStyle = "white";
        for (let x = 0; x < canvas.width / 2; x++) {
            for (let y = 0; y < canvas.height / 2; y++) {
                const random = rand();
                if (random > 0.999) {
                    context.fillRect(x * 2, y * 2, 2, 2);
                }
            }
        }
    }


    document.onscroll = function () {
        const scrollPos = window.scrollY;
        canvas.style.transform = `translateY(${scrollPos}px)`;
    };

    window.addEventListener("resize", function () {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
        draw();
    });

    draw();
}

if (!localStorage.getItem("stars")) {
    localStorage.setItem("stars", "disable");
}

if (localStorage.getItem("stars") === "enable") {
    console.log("Start stars");
    stars();
} else {
    console.log("Render static background");
    starsStatic();
}
