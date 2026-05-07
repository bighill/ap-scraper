(function () {
  "use strict";

  var statusEl = document.getElementById("status");
  var listEl = document.getElementById("list");

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

  function render(articles) {
    listEl.innerHTML = "";
    if (!articles.length) {
      setStatus("No articles yet.");
      listEl.hidden = true;
      return;
    }

    setStatus(articles.length + " article" + (articles.length === 1 ? "" : "s"));
    listEl.hidden = false;

    articles.forEach(function (a) {
      var li = document.createElement("li");
      li.className = "article-card";

      var titleA = document.createElement("a");
      titleA.className = "title";
      titleA.href = a.url || "#";
      titleA.textContent = a.title || "(no title)";
      titleA.rel = "noopener noreferrer";
      titleA.target = "_blank";
      li.appendChild(titleA);

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

      listEl.appendChild(li);
    });
  }

  function load() {
    setStatus("Loading…");
    listEl.hidden = true;

    fetch("/articles", { headers: { Accept: "application/json" } })
      .then(function (res) {
        if (!res.ok) {
          throw new Error("HTTP " + res.status);
        }
        return res.json();
      })
      .then(function (data) {
        if (!Array.isArray(data)) {
          throw new Error("Invalid response");
        }
        render(data);
      })
      .catch(function (err) {
        setStatus("Could not load articles: " + (err.message || String(err)), true);
        listEl.hidden = true;
      });
  }

  load();
})();
