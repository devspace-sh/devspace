// Turn off ESLint for this file because it's sent down to users as-is.
/* eslint-disable */
window.addEventListener('load', function() {
  var width = window.innerWidth;

  var ul = document.getElementsByClassName("nav-site")[0];
  var lis = ul.querySelectorAll("li");
  var li = lis[lis.length - 1];
  var navWrapper = document.getElementsByClassName("navigationWrapper")[0];

  window.addEventListener("resize", onResize, false);

  function onResize() {
    width = window.innerWidth;
    console.log("resize")

    var liActive = document.getElementsByClassName("hamburger-active").length;

    if(width < 801 && !liActive) {
      ul.classList += " responsive-nav";
      li.classList += "hamburger-active";

      li.addEventListener("click", onHamburgerClick)

    } else if(width > 800 && liActive) {
      ul.classList = "nav-site nav-site-internal";
      li.classList = "";
    }
  }

  function onHamburgerClick() {
    var burgerIsOpen = document.getElementsByClassName("burger-open").length;

    if(width < 801 && !burgerIsOpen) {
      navWrapper.classList += " burger-open";
      li.removeEventListener("click", onHamburgerClick)
      li.addEventListener("click", onCloseClick)
    }
  }

  function onCloseClick() {
    navWrapper.classList = "navigationWrapper navigationSlider";
    li.removeEventListener("click", onCloseClick)
    li.addEventListener("click", onHamburgerClick)
  }
});
