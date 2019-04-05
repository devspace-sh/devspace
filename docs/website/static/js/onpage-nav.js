var firstCall = true;

const highlightActiveOnPageLink = function() {
    var activeHash;

    if (firstCall) {
        firstCall = false;

        if (location.hash.length > 0) {
            activeHash = location.hash.substr(1);

            const activeAnchors = document.querySelectorAll(".anchor[id='" + activeHash + "'");

            if (activeAnchors.length > 0) {
                for (let i = 0; i < activeAnchors.length; i++) {
                    let activeAnchor = activeAnchors[i];

                    for ( ; activeAnchor && activeAnchor !== document; activeAnchor = activeAnchor.parentElement ) {
                        if (activeAnchor.tagName == "DETAILS") {
                            activeAnchor.open = true;
                        }
                    }
                }
            }
        }
        window.addEventListener('scroll', highlightActiveOnPageLink);
    }

    if (!activeHash) {
        const anchors = document.querySelectorAll("h2 > .anchor, h3 > .anchor");

        if (document.scrollingElement.scrollTop < 100 && anchors.length > 0) {
            activeHash = anchors[0].attributes.id.value;
        } else {
            for (let i = 0; i < anchors.length; i++) {
                const anchor = anchors[i];

                if (anchor.parentElement.getBoundingClientRect().top < window.screen.availHeight*0.5) {
                    activeHash = anchor.attributes.id.value;
                }
            }
        }
    
        if (!activeHash) {
            const firstOnPageNavLink = document.querySelectorAll(".toc-headings:first-child > li:first-child > a");

            if (firstOnPageNavLink.attributes) {
                activeHash = firstOnPageNavLink.attributes.href.value.substr(1);
            }
        }
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

window.addEventListener('DOMContentLoaded', highlightActiveOnPageLink);
