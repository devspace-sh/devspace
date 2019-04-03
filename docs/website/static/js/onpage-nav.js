const highlightActiveOnPageLink = function() {
    const anchors = document.querySelectorAll("h2 > .anchor, h3 > .anchor");
    var activeHash;

    if (document.scrollingElement.scrollTop < 100 && anchors.length > 0) {
        activeHash = anchors[0].attributes.id.value;
    } else {
        for (let i = 0; i < anchors.length; i++) {
            const anchor = anchors[i];

            if (anchor.parentElement.getBoundingClientRect().top < window.screen.availHeight*0.8) {
                activeHash = anchor.attributes.id.value;
            }
        }
    }
    
    if (!activeHash) {
        const firstOnPageNavLink = document.querySelectorAll(".toc-headings:first-child > li:first-child > a");
        activeHash = firstOnPageNavLink.attributes.href.value.substr(1);
    }

    const allLinks = document.querySelectorAll("a");
    
    for (let i = 0; i < allLinks.length; i++) {
        const link = allLinks[i];
        link.classList.remove("active");
    }

    const activeLinks = document.querySelectorAll("a[href='#" + activeHash + "'");

    for (let i = 0; i < activeLinks.length; i++) {
        const link = activeLinks[i];
        link.classList.add("active");
    }
};

window.addEventListener('scroll', highlightActiveOnPageLink);
window.addEventListener('DOMContentLoaded', highlightActiveOnPageLink);
