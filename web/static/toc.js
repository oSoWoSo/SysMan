document.addEventListener("DOMContentLoaded", function () {
  var toc = document.getElementById("TOC");
  if (!toc) return;

  var ICON_OPEN = "\u25c0";
  var ICON_CLOSED = "\u25ba";

  // Add toc-title
  var titleEl = document.createElement("div");
  titleEl.id = "toc-title";
  toc.insertBefore(titleEl, toc.firstChild);

  // Wrap TOC + rest of <main> in .page-wrapper / .main-content
  var main = document.querySelector("main");
  if (!main) return;

  var wrapper = document.createElement("div");
  wrapper.className = "page-wrapper";

  var siblings = Array.from(main.childNodes);
  siblings.forEach(function (s) { wrapper.appendChild(s); });
  main.appendChild(wrapper);

  var mainContent = document.createElement("div");
  mainContent.className = "main-content";
  Array.from(wrapper.childNodes).forEach(function (child) {
    if (child.id !== "TOC") mainContent.appendChild(child);
  });
  wrapper.appendChild(mainContent);

  function setCollapsed(collapsed) {
    if (collapsed) {
      toc.classList.add("collapsed");
      titleEl.innerHTML = "<span>Contents</span><span id=\"toc-toggle-icon\">" + ICON_CLOSED + "</span>";
      titleEl.style.writingMode = "vertical-rl";
      titleEl.style.marginBottom = "0";
    } else {
      toc.classList.remove("collapsed");
      titleEl.innerHTML = "<span>Contents</span><span id=\"toc-toggle-icon\">" + ICON_OPEN + "</span>";
      titleEl.style.writingMode = "";
      titleEl.style.marginBottom = "";
    }
    localStorage.setItem("toc-collapsed", collapsed);
  }

  titleEl.addEventListener("click", function () {
    setCollapsed(!toc.classList.contains("collapsed"));
  });

  // Restore state
  setCollapsed(localStorage.getItem("toc-collapsed") === "true");
});
