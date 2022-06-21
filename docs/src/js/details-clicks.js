import ExecutionEnvironment from "@docusaurus/ExecutionEnvironment";

const preserveExpansionStates = function (skipEventListener) {
  const state = new URLSearchParams(window.location.search.substring(1));

  if (document.querySelectorAll('.markdown').length == 0) {
    return setTimeout(preserveExpansionStates, 100);
  }

  document.querySelectorAll('details, .tabs-container').forEach(function (el, index) {
    const expansionKey = "x" + (el.id || index);
    const stateChangeElAll = el.querySelectorAll(':scope > summary, :scope > [role="tablist"] > *');
    const anchorLinks = el.querySelectorAll(':scope a[href="'+location.hash+'"]')
    if (anchorLinks.length > 0) {
      if (el.querySelectorAll(':scope > summary a[href="'+location.hash+'"]').length > 0) {
        el.classList.add("-contains-target-link")
      }
      state.set(expansionKey, 1);
    } else {
      el.classList.remove("-contains-target-link")
      state.delete(expansionKey);
    }

    const persistState = function (i) {
      if (Number.isInteger(i)) {
        const anchorLinks = el.querySelectorAll(':scope > summary a[href^="#"]')
        const state = new URLSearchParams(window.location.search.substring(1));
        if ((el.open && el.getAttribute("data-expandable") != "false") || el.classList.contains("tabs-container")) {
          if (anchorLinks.length == 1) {
            if (anchorLinks[0].getAttribute("href") == location.hash) {
              el.classList.add("-contains-target-link")
            }
          } else {
            state.set(expansionKey, i);
          }
        } else {
          this.classList.remove("-contains-target-link")
          state.delete(expansionKey);
        }

        let query = state.toString()
        if (query) {
          query = '?' + query.replace(/^[?]/, "")
        }

        window.history.replaceState(null, '', window.location.pathname + query + window.location.hash);
      }
    }

    if (el.getAttribute("data-preserve-state") !== "true") {
      el.setAttribute("data-preserve-state", "true")

      el.addEventListener("toggle", persistState.bind(el, 1));
      stateChangeElAll.forEach(function (stateChangeEl, i) {
        stateChangeEl.addEventListener("click", persistState.bind(stateChangeEl, i + 1))
      })

      el.querySelectorAll(':scope > summary a[href^="#"]').forEach(anchorLink => {
        anchorLink.addEventListener("click", (e) => {
          e.stopImmediatePropagation()
          e.stopPropagation()
          e.preventDefault()

          const newHash = anchorLink.getAttribute("href")

          document.querySelectorAll(".-contains-target-link").forEach(function(el) {
            if (el.querySelectorAll(':scope > summary a[href="'+newHash+'"]').length == 0) {
              el.classList.remove("-contains-target-link")
            }
          })

          let query = location.search
          if (query) {
            query = '?' + query.replace(/^[?]/, "")
          }
          window.history.replaceState(null, '', window.location.pathname + query + newHash);
          
          if (!el.hasAttribute("open")) {
            anchorLink.parentElement.click()
          }
        })
      })
    }

    if (state.get(expansionKey) && el.open != true) {
      el.open = true;
      stateChangeElAll.forEach(function (stateChangeEl, i) {
        if (state.get(expansionKey) === (i + 1).toString()) {
          stateChangeEl.click()
        }
      })
    }
  });

  setTimeout(function () {
    preserveExpansionStates();
  }, 300)
}

if (ExecutionEnvironment.canUseDOM) {
  preserveExpansionStates();
  /*
  window.addEventListener("popstate", () => {
    setTimeout(() => {
      preserveExpansionStates();
    }, 300)
  });


  window.addEventListener("hashchange", () => {
    setTimeout(() => {
      preserveExpansionStates();
    }, 300)
  });*/
  
  if (location.hash) {
    setTimeout(() => {
      location.href = location.href

      const targetEl = document.querySelector('[id="'+location.hash.substr(1)+'"]')
      if (targetEl) {
        window.scroll({
          behavior: 'smooth',
          left: 0,
          top: targetEl.getBoundingClientRect().top + window.scrollY - 120
        });
      }
    }, 1000)
  }
}
