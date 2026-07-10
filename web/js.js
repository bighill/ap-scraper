(function () {
  "use strict";

  var listViewEl = document.getElementById("list-view");
  var listEl = document.getElementById("list");
  var detailViewEl = document.getElementById("detail-view");
  var detailTitleEl = document.getElementById("detail-title");
  var detailMetaEl = document.getElementById("detail-meta");
  var detailImageEl = document.getElementById("detail-image");
  var detailContentEl = document.getElementById("detail-content");
  var detailExternalEl = document.getElementById("detail-external");
  var backBtn = document.getElementById("back-btn");
  var controlsEl = document.getElementById("controls");
  var showVisibleBtn = document.getElementById("show-visible");
  var showHiddenBtn = document.getElementById("show-hidden");
  var showImagesEl = document.getElementById("show-images");

  var viewMode = "visible"; // 'visible' | 'hidden'
  var showImages = true;
  var currentArticles = [];

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
        showVisibleBtn.textContent = visible + " visible";
        showHiddenBtn.textContent = hidden + " hidden";
        controlsEl.hidden = false;
      })
      .catch(function () {
        showVisibleBtn.textContent = "";
        showHiddenBtn.textContent = "";
      });
  }

  function removeArticle(li) {
    li.parentNode.removeChild(li);
    if (!listEl.children.length) {
      listEl.hidden = true;
    }
  }

  function applyShowImages(show) {
    showImages = show;
    if (showImagesEl) {
      showImagesEl.checked = show;
    }
  }

  function loadShowImages() {
    return fetch("/settings/images", { headers: { Accept: "application/json" } })
      .then(function (res) {
        if (!res.ok) throw new Error("HTTP " + res.status);
        return res.json();
      })
      .then(function (data) {
        if (data && typeof data.show_images === "boolean") {
          applyShowImages(data.show_images);
        }
      });
  }

  function showList() {
    detailViewEl.hidden = true;
    listViewEl.hidden = false;
  }

  function showDetail(article) {
    listViewEl.hidden = true;
    detailViewEl.hidden = false;

    detailTitleEl.textContent = article.title || "(no title)";
    detailExternalEl.href = article.url || "#";
    detailMetaEl.textContent = article.posted_at ? "Posted " + formatDate(article.posted_at) : "";

    if (showImages && article.image_url) {
      detailImageEl.src = article.image_url;
      detailImageEl.hidden = false;
    } else {
      detailImageEl.hidden = true;
      detailImageEl.src = "";
    }

    detailContentEl.innerHTML = "";
    if (article.content) {
      article.content.split("\n\n").forEach(function (para) {
        var p = document.createElement("p");
        p.textContent = para;
        detailContentEl.appendChild(p);
      });
      window.scrollTo(0, 0);
    } else if (article.content_scraped_at) {
      var p = document.createElement("p");
      p.className = "article-blurb";
      p.textContent = "No article text is available for this page.";
      detailContentEl.appendChild(p);
      window.scrollTo(0, 0);
    } else if (typeof article.content_scraped_at === "undefined") {
      var p = document.createElement("p");
      p.className = "article-blurb";
      p.textContent = "Loading article text…";
      detailContentEl.appendChild(p);

      fetch("/articles/" + encodeURIComponent(article.id), {
        headers: { Accept: "application/json" },
      })
        .then(function (res) {
          if (!res.ok) throw new Error("HTTP " + res.status);
          return res.json();
        })
        .then(function (data) {
          article.content = data.content || "";
          article.content_scraped_at = data.content_scraped_at || 0;
          showDetail(article);
        })
        .catch(function (err) {
          detailContentEl.innerHTML = "";
          var p = document.createElement("p");
          p.className = "article-blurb";
          p.textContent = "Could not load article: " + (err.message || String(err));
          detailContentEl.appendChild(p);
        });
    } else {
      var p = document.createElement("p");
      p.className = "article-blurb";
      p.textContent = "Article text has not been loaded yet. It will appear after the next scheduled content scrape.";
      detailContentEl.appendChild(p);
      window.scrollTo(0, 0);
    }
  }

  function render(articles) {
    currentArticles = articles;
    listEl.innerHTML = "";
    if (!articles || !articles.length) {
      currentArticles = [];
      listEl.hidden = true;
      return;
    }

    listEl.hidden = false;

    articles.forEach(function (a) {
      var li = document.createElement("li");
      li.className = "article-card";

      var titleBtn = document.createElement("button");
      titleBtn.className = "title";
      titleBtn.type = "button";
      titleBtn.textContent = a.title || "(no title)";
      li.appendChild(titleBtn);

      var actionBtn = document.createElement("button");
      actionBtn.type = "button";
      if (viewMode === "hidden") {
        actionBtn.className = "unhide-btn";
        actionBtn.textContent = "Unhide";
      } else {
        actionBtn.className = "hide-btn";
        actionBtn.textContent = "Hide";
      }

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

      if (showImages && a.image_url) {
        var img = document.createElement("img");
        img.className = "article-thumb";
        img.src = a.image_url;
        img.alt = "";
        img.loading = "lazy";
        li.appendChild(img);
      }

      li.appendChild(actionBtn);

      titleBtn.addEventListener("click", function () {
        showDetail(a);
      });

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
            console.error("Failed to update article:", err.message || String(err));
          });
      });

      listEl.appendChild(li);
    });
  }

  function load() {
    listEl.hidden = true;
    showList();

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
        console.error("Could not load articles:", err.message || String(err));
        listEl.hidden = true;
      });
  }

  function setActiveView() {
    showVisibleBtn.classList.toggle("is-active", viewMode === "visible");
    showHiddenBtn.classList.toggle("is-active", viewMode === "hidden");
  }

  showVisibleBtn.addEventListener("click", function () {
    if (viewMode === "visible") return;
    viewMode = "visible";
    setActiveView();
    load();
  });

  showHiddenBtn.addEventListener("click", function () {
    if (viewMode === "hidden") return;
    viewMode = "hidden";
    setActiveView();
    load();
  });

  setActiveView();

  backBtn.addEventListener("click", function () {
    showList();
  });

  showImagesEl.addEventListener("change", function () {
    var newVal = showImagesEl.checked;
    fetch("/settings/images", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ show_images: newVal }),
    })
      .then(function (res) {
        if (!res.ok) throw new Error("HTTP " + res.status);
        applyShowImages(newVal);
        render(currentArticles);
      })
      .catch(function (err) {
        console.error("Failed to update images setting:", err.message || String(err));
        applyShowImages(!newVal);
      });
  });

  loadShowImages().then(load).catch(load);
})();
