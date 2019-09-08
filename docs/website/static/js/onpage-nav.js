var firstCall = true;

const highlightDetailsOnActiveHash = function(activeHash, doNotOpen) {
    const activeAnchors = document.querySelectorAll(".anchor[id='" + activeHash + "'");
    const detailsElements = document.querySelectorAll("details");

    for (let i = 0; i < detailsElements.length; i++) {
        let detailsElement = detailsElements[i];

        detailsElement.classList.remove("active");
    }

    if (activeAnchors.length > 0) {
        for (let i = 0; i < activeAnchors.length; i++) {
            let element = activeAnchors[i];

            for ( ; element && element !== document; element = element.parentElement ) {
                if (element.tagName == "DETAILS") {
                    element.classList.add("active");

                    if (!doNotOpen) {
                        element.open = true;
                    }
                }
            }
        }
    }
};

const highlightActiveOnPageLink = function() {
    var activeHash;

    if (firstCall) {
        firstCall = false;

        if (location.hash.length > 0) {
            activeHash = location.hash.substr(1);

            highlightDetailsOnActiveHash(activeHash);
        }
        window.addEventListener('scroll', highlightActiveOnPageLink);
    }

    setTimeout(function() {
        if (!activeHash) {
            const anchors = document.querySelectorAll("h2 > .anchor, h3 > .anchor");
    
            if (document.scrollingElement.scrollTop < 100 && anchors.length > 0) {
                activeHash = anchors[0].attributes.id.value;
            } else {
                for (let i = 0; i < anchors.length; i++) {
                    const anchor = anchors[i];
    
                    if (anchor.parentElement.getBoundingClientRect().top < window.screen.availHeight*0.7) {
                        activeHash = anchor.attributes.id.value;
                    } else {
                        break;
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
    }, 100)
};

const hashLinkClickSet = false;

const allowHashLinkClick = function() {
    if (!hashLinkClickSet) {
        const hashLinkIcons = document.querySelectorAll(".hash-link-icon");
        
        for (let i = 0; i < hashLinkIcons.length; i++) {
            const hashLinkIcon = hashLinkIcons[i];
            hashLinkIcon.addEventListener("mousedown", function() {
                history.pushState(null, null, hashLinkIcon.parentElement.attributes.href.value);
                highlightActiveOnPageLink();
                highlightDetailsOnActiveHash(location.hash.substr(1), true);
            });
        }
    }
};

window.addEventListener('DOMContentLoaded', allowHashLinkClick);
window.addEventListener('DOMContentLoaded', highlightActiveOnPageLink);
window.addEventListener('popstate', function (event) {
    highlightDetailsOnActiveHash(location.hash.substr(1));
}, false);
