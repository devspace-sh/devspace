document.addEventListener("DOMContentLoaded", function() {
    var starButton = document.querySelector(".star-button");
    var starButtonParent = document.querySelector(".headerWrapper > header");

    if (starButton) {
        starButtonParent.insertBefore(starButton, starButtonParent.children[0]);
    }
});
