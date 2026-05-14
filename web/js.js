(function () {
  "use strict";

  var statusEl = document.getElementById("status");
  var listEl = document.getElementById("list");
  var controlsEl = document.getElementById("controls");
  var countBadgeEl = document.getElementById("count-badge");
  var toggleViewEl = document.getElementById("toggle-view");

  var viewMode = "visible"; // 'visible' | 'hidden'

  function setStatus(text, isError) {
    statusEl.textContent = text;
    statusEl.classList.toggle("is-error", !!isError);
  }

  function formatDate(ms) {
    if (ms == null || ms === 0) {
      return "";
    }
    try {
      return new Date(ms).toLocaleString(undefined, {
        dateStyle: "medium",
        timeStyle: "short",
      });
    } catch (_) {
      return "";
    }
  }

  function updateCounts() {
    fetch("/articles/count", { headers: { Accept: "application/json" } })
      .then(function (res) {
        if (!res.ok) throw new Error("HTTP " + res.status);
        return res.json();
      })
      .then(function (data) {
        var hidden = data.hidden || 0;
        var visible = data.visible || 0;
        countBadgeEl.textContent = hidden + " hidden / " + visible + " visible";
        controlsEl.hidden = false;
      })
      .catch(function () {
        countBadgeEl.textContent = "";
      });
  }

  function removeArticle(li) {
    var next = li.nextElementSibling;
    var prev = li.previousElementSibling;
    li.parentNode.removeChild(li);
    if (next && prev) {
      // re-assert borders via CSS sibling selector; no-op for JS.
    }
    if (!listEl.children.length) {
      setStatus(viewMode === "hidden" ? "No hidden articles." : "No articles.");
      listEl.hidden = true;
    } else {
      setStatus(listEl.children.length + " article" + (listEl.children.length === 1 ? "" : "s"));
    }
  }

  function render(articles) {
    listEl.innerHTML = "";
    if (!articles || !articles.length) {
      setStatus(viewMode === "hidden" ? "No hidden articles." : "No articles yet.");
      listEl.hidden = true;
      return;
    }

    setStatus(articles.length + " article" + (articles.length === 1 ? "" : "s"));
    listEl.hidden = false;

    articles.forEach(function (a) {
      var li = document.createElement("li");
      li.className = "article-card";

      var header = document.createElement("div");
      header.className = "article-header";

      var titleA = document.createElement("a");
      titleA.className = "title";
      titleA.href = a.url || "#";
      titleA.textContent = a.title || "(no title)";
      titleA.rel = "noopener noreferrer";
      titleA.target = "_blank";
      header.appendChild(titleA);

      var actionBtn = document.createElement("button");
      actionBtn.type = "button";
      if (viewMode === "hidden") {
        actionBtn.className = "unhide-btn";
        actionBtn.textContent = "Unhide";
      } else {
        actionBtn.className = "hide-btn";
        actionBtn.textContent = "Hide";
      }
      header.appendChild(actionBtn);

      li.appendChild(header);

      var posted = formatDate(a.posted_at);
      if (posted) {
        var meta = document.createElement("p");
        meta.className = "article-meta";
        meta.textContent = "Posted " + posted;
        li.appendChild(meta);
      }

      if (a.blurb) {
        var blurb = document.createElement("p");
        blurb.className = "article-blurb";
        blurb.textContent = a.blurb;
        li.appendChild(blurb);
      }

      if (a.image_url) {
        var img = document.createElement("img");
        img.className = "article-thumb";
        img.src = a.image_url;
        img.alt = "";
        img.loading = "lazy";
        li.appendChild(img);
      }

      actionBtn.addEventListener("click", function () {
        var url = a.url;
        var endpoint = viewMode === "hidden" ? "/articles/unhide" : "/articles/hide";
        fetch(endpoint, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ url: url }),
        })
          .then(function (res) {
            if (!res.ok) throw new Error("HTTP " + res.status);
            removeArticle(li);
            updateCounts();
          })
          .catch(function (err) {
            setStatus("Failed to update article: " + (err.message || String(err)), true);
          });
      });

      listEl.appendChild(li);
    });
  }

  function load() {
    setStatus("Loading…");
    listEl.hidden = true;

    var url = viewMode === "hidden" ? "/articles?hidden=1" : "/articles";
    fetch(url, { headers: { Accept: "application/json" } })
      .then(function (res) {
        if (!res.ok) throw new Error("HTTP " + res.status);
        return res.json();
      })
      .then(function (data) {
        if (!Array.isArray(data)) throw new Error("Invalid response");
        render(data);
        updateCounts();
      })
      .catch(function (err) {
        setStatus("Could not load articles: " + (err.message || String(err)), true);
        listEl.hidden = true;
      });
  }

  toggleViewEl.addEventListener("click", function () {
    viewMode = viewMode === "visible" ? "hidden" : "visible";
    toggleViewEl.textContent = viewMode === "visible" ? "Show hidden" : "Show main";
    load();
  });

  load();
})();
